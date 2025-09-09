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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

const (
	MaxWaitDuration = 5 * time.Second
	jdtlsVersion    = "1.39.0-202408291433"
	jdtlsURL        = "https://download.eclipse.org/jdtls/milestones/1.39.0/jdt-language-server-1.39.0-202408291433.tar.gz"
)

// untar takes a destination path and a reader; a tar reader loops over the tar file
// and writes each file to the destination path.
func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			// make sure the directory for the file exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

func setupJDTLS() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	javaDir := filepath.Dir(currentFile)
	installDir := filepath.Join(javaDir, "lsp", "jdtls")

	// Check for any existing JDTLS installation
	existingDirs, err := filepath.Glob(filepath.Join(installDir, "jdt-language-server-*"))
	if err == nil && len(existingDirs) > 0 {
		for _, dir := range existingDirs {
			info, err := os.Stat(dir)
			if err == nil && info.IsDir() {
				// Check if launcher jar exists in this directory
				launcherPattern := filepath.Join(dir, "plugins", "org.eclipse.equinox.launcher_*.jar")
				matches, err := filepath.Glob(launcherPattern)
				if err == nil && len(matches) > 0 {
					log.Printf("Found existing JDT Language Server at %s. Skipping installation.", dir)
					return dir, nil
				}
			}
		}
	}

	log.Printf("JDT Language Server not found locally. Downloading and installing version %s...", jdtlsVersion)
	jdtlsDir := filepath.Join(installDir, "jdt-language-server-"+jdtlsVersion)

	// Create download directory
	downloadDir := filepath.Join(installDir, "download")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	// Download
	tarballName := "jdt-language-server-" + jdtlsVersion + ".tar.gz"
	tarballPath := filepath.Join(downloadDir, tarballName)
	log.Printf("Downloading from %s...", jdtlsURL)
	resp, err := http.Get(jdtlsURL)
	if err != nil {
		return "", fmt.Errorf("failed to download JDTLS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download JDTLS: received status code %d", resp.StatusCode)
	}

	out, err := os.Create(tarballPath)
	if err != nil {
		return "", fmt.Errorf("failed to create tarball file: %w", err)
	}
	//defer os.Remove(tarballPath) // Clean up tarball after function returns

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		out.Close()
		return "", fmt.Errorf("failed to save tarball: %w", err)
	}
	out.Close() // Close file before untarring

	// Extract
	log.Printf("Extracting to %s...", installDir)
	file, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	if err := untar(jdtlsDir, file); err != nil {
		return "", fmt.Errorf("failed to extract JDTLS: %w", err)
	}

	log.Printf("JDT Language Server installed successfully in %s.", jdtlsDir)
	return jdtlsDir, nil
}

func GetDefaultLSP(LspOptions map[string]string) (lang uniast.Language, name string) {
	return uniast.Java, generateExecuteCmd(LspOptions)
}

func generateExecuteCmd(LspOptions map[string]string) string {
	var jdtRootPATH string
	// First, check environment variable
	if envPath := os.Getenv("JDTLS_ROOT_PATH"); len(envPath) != 0 {
		jdtRootPATH = envPath
		log.Printf("Using JDTLS_ROOT_PATH from environment: %s", jdtRootPATH)
	} else {
		// If env var is not set, run auto-setup
		var err error
		jdtRootPATH, err = setupJDTLS()
		if err != nil {
			panic(fmt.Sprintf("Failed to setup JDT Language Server: %v", err))
		}
	}

	// Get the absolute path to the current file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current file path")
	}
	javaDir := filepath.Dir(currentFile)

	// Find launcher jar
	launcherPattern := filepath.Join(jdtRootPATH, "plugins", "org.eclipse.equinox.launcher_*.jar")
	matches, err := filepath.Glob(launcherPattern)
	if err != nil || len(matches) == 0 {
		panic(fmt.Sprintf("Could not find org.eclipse.equinox.launcher_*.jar in %s/plugins", jdtRootPATH))
	}
	jdtLsPath := matches[0]

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
	configPath := filepath.Join(jdtRootPATH, configDir)
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
	javaCmd := "java "
	if len(LspOptions["java.home"]) != 0 {
		javaCmd = LspOptions["java.home"] + " "
	}
	return javaCmd + strings.Join(args, " ")
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
