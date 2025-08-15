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

package java

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

const MaxWaitDuration = 5 * time.Second

func GetDefaultLSP() (lang uniast.Language, name string) {

	return uniast.Java, generateExecuteCmd()
}

func generateExecuteCmd() string {
	// Get the absolute path to the current file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current file path")
	}
	javaDir := filepath.Dir(currentFile)
	jdtLsPath := filepath.Join(javaDir, "lsp", "jdtls", "jdt-language-server-1.39.0-202408291433", "plugins", "org.eclipse.equinox.launcher_1.6.900.v20240613-2009.jar")
	// Determine the configuration path based on OS and architecture
	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "mac"
	case "windows":
		osName = "win"
	default:
		osName = runtime.GOOS
	}
	configDir := fmt.Sprintf("config_%s", osName)
	if runtime.GOARCH == "arm64" {
		configDir += "_arm"
	}
	configPath := filepath.Join(javaDir, "lsp", "jdtls", "jdt-language-server-1.39.0-202408291433", configDir)
	dataPath := filepath.Join(javaDir, "lsp", "jdtls", "runtime")
	args := []string{
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-Dlog.level=ALL",
		"-noverify",
		"-Xmx1G",
		fmt.Sprintf("-jar %s", jdtLsPath),
		fmt.Sprintf("-configuration %s", configPath),
		fmt.Sprintf("-data %s", dataPath),
		"--add-modules=ALL-SYSTEM",
		"--add-opens java.base/java.util=ALL-UNNAMED",
		"--add-opens java.base/java.lang=ALL-UNNAMED",
	}
	return "java " + strings.Join(args, " ")
}

func CheckRepo(repo string) (string, time.Duration) {
	openfile := ""

	// Give the LSP sometime to initialize
	_, size := utils.CountFiles(repo, ".java", "SKIPDIR")
	wait := 2*time.Second + time.Second*time.Duration(size/1024)
	if wait > MaxWaitDuration {
		wait = MaxWaitDuration
	}
	return openfile, wait
}
