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
	"os"
	"os/exec"
	"path/filepath"
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

func pylspInstallPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(cacheDir, "abcoder", lspPath), nil
}

func validatePylspSource(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect cached pylsp path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cached pylsp path %q is not a directory", path)
	}
	gitInfo, gitErr := os.Stat(filepath.Join(path, ".git"))
	if gitErr != nil || !gitInfo.IsDir() {
		return fmt.Errorf("cached pylsp path %q is not a git checkout", path)
	}
	projectInfo, projectErr := os.Stat(filepath.Join(path, "pyproject.toml"))
	if projectErr != nil || projectInfo.IsDir() {
		return fmt.Errorf("cached pylsp checkout %q has no pyproject.toml", path)
	}
	return nil
}

func clonePylspSource(path string) error {
	log.Error("Installing pylsp... Now running git clone -b %s %s %s", lspBranch, lspUrl, path)
	if output, err := exec.Command("git", "clone", "-b", lspBranch, lspUrl, path).CombinedOutput(); err != nil {
		return fmt.Errorf("clone pylsp into %q: %w: %s", path, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func ensurePylspSourceWithClone(path string, clone func(string) error) error {
	if _, err := os.Stat(path); err == nil {
		if err := validatePylspSource(path); err != nil {
			return err
		}
		log.Info("Reusing cached pylsp checkout at %s", path)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect cached pylsp path %q: %w", path, err)
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create pylsp cache directory: %w", err)
	}
	tempPath, err := os.MkdirTemp(parent, ".pylsp-clone-")
	if err != nil {
		return fmt.Errorf("create temporary pylsp checkout: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempPath); err != nil {
			log.Error("failed to clean up temporary pylsp checkout %s: %v", tempPath, err)
		}
	}()

	if err := clone(tempPath); err != nil {
		return err
	}
	if err := validatePylspSource(tempPath); err != nil {
		return fmt.Errorf("validate cloned pylsp checkout: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if validationErr := validatePylspSource(path); validationErr == nil {
			log.Info("Reusing pylsp checkout installed concurrently at %s", path)
			return nil
		}
		return fmt.Errorf("publish pylsp checkout at %q: %w", path, err)
	}
	return nil
}

func ensurePylspSource(path string) error {
	return ensurePylspSourceWithClone(path, clonePylspSource)
}

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
		log.Info("pylsp already installed: %v", string(out))
		return lspName, nil
	}
	if err := CheckPythonVersion(); err != nil {
		log.Error("python version check failed: %v", err)
		return "", err
	}
	path, err := pylspInstallPath()
	if err != nil {
		log.Error("failed to determine pylsp install path: %v", err)
		return "", err
	}
	if err := ensurePylspSource(path); err != nil {
		log.Error("failed to prepare pylsp source: %v", err)
		return "", err
	}
	// Install the cached checkout in editable mode.
	log.Error("Installing pylsp via pip. This might take some time, make sure the network connection is ok.")
	if err := exec.Command("python3", "-m", "pip", "install", "--break-system-packages", "-e", path).Run(); err != nil {
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
