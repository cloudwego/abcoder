/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lang

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/cxx"
	"github.com/cloudwego/abcoder/lang/golang/parser"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/python"
	"github.com/cloudwego/abcoder/lang/rust"
	"github.com/cloudwego/abcoder/lang/typescript"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// ParseOptions is the options for parsing the repo.
type ParseOptions struct {
	// LSP sever executable path
	LSP string
	// Language of the repo
	Verbose bool
	collect.CollectOption
	// specify the repo id
	RepoID string
	// 输出路径
	OutputPath string
	// TS options
	// tsconfig string
	TSParseOptions
}

type TSParseOptions struct {
	// tsconfig path
	TSConfig string
	// srcDir path
	TSSrcDir []string
}

func Parse(ctx context.Context, uri string, args ParseOptions) ([]byte, error) {
	if !filepath.IsAbs(uri) {
		uri, _ = filepath.Abs(uri)
	}

	// Handle TypeScript separately
	if args.Language == uniast.TypeScript {
		return parseTSProject(ctx, uri, args)
	}

	l, lspPath, err := checkLSP(args.Language, args.LSP)
	if err != nil {
		return nil, err
	}
	openfile, opentime, err := checkRepoPath(uri, l)
	if err != nil {
		return nil, err
	}

	var client *lsp.LSPClient
	if lspPath != "" {
		// Initialize the LSP client
		log.Info("start initialize LSP server %s...\n", lspPath)
		var err error
		client, err = lsp.NewLSPClient(uri, openfile, opentime, lsp.ClientOptions{
			Server:   lspPath,
			Language: l,
			Verbose:  args.Verbose,
		})
		if err != nil {
			log.Error("failed to initialize LSP server: %v\n", err)
			return nil, err
		}
		log.Info("end initialize LSP server")
	}

	repo, err := collectSymbol(ctx, client, uri, args.CollectOption)
	if err != nil {
		log.Error("Failed to collect symbols: %v\n", err)
		return nil, err
	}
	log.Info("all symbols collected, start writing to stdout...\n")

	if args.RepoID != "" {
		repo.Name = args.RepoID
	}

	repo.ASTVersion = uniast.Version

	out, err := json.Marshal(repo)
	if err != nil {
		log.Error("Failed to marshal repository: %v\n", err)
		return nil, err
	}
	return out, nil
}

func checkRepoPath(repoPath string, language uniast.Language) (openfile string, wait time.Duration, err error) {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("repository not found: %s", repoPath)
	}
	switch language {
	case uniast.Rust:
		// NOTICE: open the Cargo.toml file is required for Rust projects
		openfile, wait = rust.CheckRepo(repoPath)
	case uniast.Cxx:
		openfile, wait = cxx.CheckRepo(repoPath)
	case uniast.Python:
		openfile, wait = python.CheckRepo(repoPath)
	case uniast.TypeScript:
		openfile, wait = typescript.CheckRepo(repoPath)
	default:
		openfile = ""
		wait = 0
	}

	log.Info("open file '%s' and wait for %d seconds for initialize workspace\n", openfile, wait/time.Second)
	return
}

func checkLSP(language uniast.Language, lspPath string) (l uniast.Language, s string, err error) {
	switch language {
	case uniast.Rust:
		l, s = rust.GetDefaultLSP()
	case uniast.Cxx:
		l, s = cxx.GetDefaultLSP()
	case uniast.Python:
		l, s = python.GetDefaultLSP()
	case uniast.TypeScript:
		l, s = typescript.GetDefaultLSP()
	case uniast.Golang:
		l = uniast.Golang
		s = ""
		if _, err := exec.LookPath("go"); err != nil {
			if _, err := os.Stat(lspPath); os.IsNotExist(err) {
				log.Error("Go compiler not found, please make it excutable!\n", lspPath)
				return uniast.Unknown, "", err
			}
		}
		return
	default:
		return uniast.Unknown, "", fmt.Errorf("unsupported language: %s", language)
	}
	// check if lsp excutable
	if lspPath != "" {
		if _, err := exec.LookPath(lspPath); err != nil {
			if _, err := os.Stat(lspPath); os.IsNotExist(err) {
				log.Error("Language server %s not found, please make it excutable!\n", lspPath)
				return uniast.Unknown, "", err
			}
		}
		s = lspPath
	}

	return
}

func collectSymbol(ctx context.Context, cli *lsp.LSPClient, repoPath string, opts collect.CollectOption) (repo *uniast.Repository, err error) {
	if opts.Language == uniast.Golang {
		repo, err = callGoParser(ctx, repoPath, opts)
		if err != nil {
			return nil, err
		}
	} else {
		collector := collect.NewCollector(repoPath, cli)
		collector.CollectOption = opts
		log.Info("start collecting symbols...\n")
		err = collector.Collect(ctx)
		if err != nil {
			return nil, err
		}
		log.Info("all symbols collected.\n")
		log.Info("start exporting symbols...\n")
		repo, err = collector.Export(ctx)
		if err != nil {
			return nil, err
		}
	}

	if err := repo.BuildGraph(); err != nil {
		return nil, err
	}
	return repo, nil
}

func callGoParser(ctx context.Context, repoPath string, opts collect.CollectOption) (*uniast.Repository, error) {
	goopts := parser.Options{}
	if opts.LoadExternalSymbol {
		goopts.ReferCodeDepth = 1
	}
	if !opts.NoNeedComment {
		goopts.CollectComment = true
	}
	if !opts.NotNeedTest {
		goopts.NeedTest = true
	}
	if opts.LoadByPackages {
		goopts.LoadByPackages = true
	}
	goopts.Excludes = opts.Excludes
	p := parser.NewParser(repoPath, repoPath, goopts)
	repo, err := p.ParseRepo()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func parseTSProject(ctx context.Context, repoPath string, opts ParseOptions) ([]byte, error) {
	parserPath, err := exec.LookPath("abcoder-ts-parser")
	if err != nil {
		log.Info("abcoder-ts-parser not found, installing...")
		cmd := exec.Command("npm", "install", "-g", "abcoder-ts-parser")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to install abcoder-ts-parser: %v", err)
		}
		parserPath, err = exec.LookPath("abcoder-ts-parser")
		if err != nil {
			return nil, fmt.Errorf("failed to find abcoder-ts-parser after installation: %v", err)
		}
	}

	args := []string{"parse", repoPath}
	if len(opts.TSSrcDir) > 0 {
		args = append(args, "--src", strings.Join(opts.TSSrcDir, ","))
	}
	if opts.TSConfig != "" {
		args = append(args, "--tsconfig", opts.TSConfig)
	}

	// Use a temporary file for output since we need to return the content
	tempFile, err := os.CreateTemp("", "abcoder-ts-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	args = append(args, "--output", tempFile.Name())

	cmd := exec.CommandContext(ctx, parserPath, args...)
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=65536")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	log.Info("Running abcoder-ts-parser with args: %v", args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("abcoder-ts-parser failed: %v, stderr: %s", err, stderr.String())
	}

	// Read the output from the temp file
	output, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read parser output: %v", err)
	}

	return output, nil
}
