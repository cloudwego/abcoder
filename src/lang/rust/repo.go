// Copyright 2025 CloudWeGo Authors
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     https://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/lsp"
	"github.com/cloudwego/abcoder/src/lang/utils"
)

const MaxWaitDuration = 5 * time.Minute

func CheckRepo(repo string) (string, time.Duration) {
	// NOTICE: open the Cargo.toml file is required for Rust projects
	openfile := utils.FirstFile(repo, ".rs", filepath.Join(repo, "target"))

	// check if compiling cache exist
	if _, err := os.Stat(filepath.Join(repo, "target")); os.IsNotExist(err) {
		log.Info("Compiling cache not found, run `cargo build` to compile the project first...\n")
		// compile with the default version first
		if err := RunCmdInDir(repo, "cargo", "build"); err == nil {
			goto next
		}
		// update the toolchain and recompile
		log.Info("Compiling faield, update the rust toolchain...\n")
		if err := UpdateToolChain(repo, 27); err != nil {
			log.Error("Failed to update the rust toolchain: %v\n", err)
			os.Exit(1)
		}
		log.Info("Recompile the project for second time...\n")
		if err := RunCmdInDir(repo, "cargo", "build"); err == nil {
			goto next
		}
		log.Error("Failed to compile the project, update the rust toolchain to the last commit date\n", err)
		if err := UpdateToolChain(repo, -1); err != nil {
			log.Error("Failed to update the rust toolchain: %v\n", err)
			os.Exit(1)
		}
		log.Info("Recompile the project for third time...\n")
		if err := RunCmdInDir(repo, "cargo", "build"); err == nil {
			goto next
		}
		log.Error("Failed to compile the project, please check the project\n")
		os.Exit(1)
	}

next:
	// NOTICE: wait for Rust projects based on code files
	_, size := utils.CountFiles(repo, ".rs", "./target")
	wait := 15*time.Second + time.Second*time.Duration(size/1024)
	if wait > MaxWaitDuration {
		wait = MaxWaitDuration
	}
	return openfile, wait
}

func GetDefaultLSP() (lang lsp.Language, name string) {
	return lsp.Rust, "rust-analyzer"
}

func GetLastCommitTime(repo string) time.Time {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%ct")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		log.Error("Failed to get last commit time: %v\n", err)
		os.Exit(1)
	}
	commitTimeInt, err := strconv.ParseInt(string(out), 10, 64)
	if err != nil {
		log.Error("Failed to parse commit time: %v\n", err)
		os.Exit(1)
	}
	commitTime := time.Unix(commitTimeInt, 0)
	return commitTime
}

// call `rustup` to install the latest Rust toolchain on the end of month
func UpdateToolChain(repo string, recommandDay int) error {
	if _, err := os.ReadFile(filepath.Join(repo, "rut-toolchain.toml")); err != nil {

		// must be stable, just update to the latest
		if err := RunCmdInDir(repo, "rustup", "update"); err != nil {
			log.Error("Failed to update rust toolchain: %v\n", err)
			os.Exit(1)
		}

	} else {
		// must be nightly, need specific version

		// get the last commit time
		date := GetLastCommitTime(repo)
		// update the day, avoid install too many versions
		if recommandDay > 0 && recommandDay < 28 {
			date = time.Date(date.Year(), date.Month(), recommandDay, 0, 0, 0, 0, time.Local)
		}
		version := fmt.Sprintf("nightly-%s", date.Format("2006-01-02"))
		if err := RunCmdInDir(repo, "rustup", "toolchain", "install", version); err != nil {
			log.Error("Failed to install rust toolchain: %v\n", err)
			return err
		}
		// override
		if err := RunCmdInDir(repo, "rustup", "override", "set", version); err != nil {
			log.Error("Failed to set default rust toolchain: %v\n", err)
			return err
		}
	}

	// add rust-analyzer
	if err := RunCmdInDir(repo, "rustup", "component", "add", "rust-analyzer"); err != nil {
		log.Error("Failed to install rust-analyzer: %v\n", err)
		return err
	}
	return nil
}

func RunCmdInDir(dir string, cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Dir = dir
	log.Info_skip(1, "Run command `%v` in %s\n", command, dir)
	return command.Run()
}
