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

package python

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

const MaxWaitDuration = 5 * time.Second
const lspName = "pylsp"
const lspUrl = "https://github.com/Hoblovski/python-lsp-server.git"
const lspBranch = "abc"
const lspPath = "pylsp"

func CheckPythonVersion() error {
	// Check python3 command availability and get version.
	output, err := exec.Command("python3", "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("python3 not found: %w. Do you have it installed? Or is it `python` but not aliased?", err)
	}

	// The regex is corrected to handle a capital 'P' and correctly capture the minor version.
	format := `^Python 3\.(\d+)\..*$`
	ptn := regexp.MustCompile(format)
	matches := ptn.FindStringSubmatch(strings.TrimSpace(string(output)))
	if len(matches) < 2 {
		return fmt.Errorf("unexpected `python3 --version` output format: %q", output)
	}
	subver, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse python version from `python3 --version` output %q: %w", output, err)
	}
	if subver < 9 {
		return fmt.Errorf("python version 3.%d is not supported; 3.9 or higher is required", subver)
	}
	return nil
}

func InstallLanguageServer() (string, error) {
	if out, err := exec.Command("pylsp", "--version").CombinedOutput(); err == nil {
		log.Info("pylsp already installed: %v", out)
		return lspName, nil
	}
	if err := CheckPythonVersion(); err != nil {
		log.Error("python version check failed: %v", err)
		return "", err
	}
	// git clone
	log.Error("Installing pylsp... Now running git clone -b %s %s %s", lspBranch, lspUrl, lspPath)
	if err := exec.Command("git", "clone", "-b", lspBranch, lspUrl, lspPath).Run(); err != nil {
		log.Error("git clone failed: %v", err)
		return "", err
	}
	// python -m pip install -e projectRoot/pylsp
	log.Error("Installing pylsp via pip. This might take some time, make sure the network connection is ok.")
	if err := exec.Command("python3", "-m", "pip", "install", "--break-system-packages", "-e", lspPath).Run(); err != nil {
		log.Error("python3 -m pip install failed: %v", err)
		return "", err
	}
	if err := exec.Command("pylsp", "--version").Run(); err != nil {
		log.Error("`pylsp --version` failed: %v", err)
		return "", err
	}
	log.Error("pylsp installed.")
	return lspName, nil
}

func GetDefaultLSP() (lang uniast.Language, name string) {
	InstallLanguageServer()
	return uniast.Python, lspName
}

func CheckRepo(repo string) (string, time.Duration) {
	openfile := ""

	// Give the LSP sometime to initialize
	_, size := utils.CountFiles(repo, ".py", "SKIPDIR")
	wait := 2*time.Second + time.Second*time.Duration(size/1024)
	if wait > MaxWaitDuration {
		wait = MaxWaitDuration
	}
	return openfile, wait
}
