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

/**
 * Copyright 2024 ByteDance Inc.
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

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudwego/abcoder/src/lang/collect"
	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/lsp"
	"github.com/cloudwego/abcoder/src/lang/rust"
	"github.com/cloudwego/abcoder/src/lang/uniast"
	"github.com/spf13/cobra"
)

func main() {
	var client *lsp.LSPClient
	var repoPath string
	var lang lsp.Language

	// Define flags
	var flagLsp string
	var flagVerbose, flagDebug bool
	var loadExternalSymbol bool
	var NoNeedComment bool
	var excludes []string

	var rootCmd = &cobra.Command{
		Use: `lang <Action> <Language> <RepoPath>
Action:
   collect		Parse repo and export AST
Language:
   rust			For rust codes`,
		Short: "Lang: An universal language analyzer based on Language-Server-Protocol",
		Args:  cobra.ExactArgs(3),

		PreRun: func(cmd *cobra.Command, args []string) {
			// validate arguments
			checkVerbose(flagVerbose, flagDebug)
			var err error
			repoPath, err = filepath.Abs(args[2])
			if err != nil {
				log.Error("Failed to get absolute path of repository: %v\n", err)
				os.Exit(1)
			}
			l, lspPath := checkLSP(args[1], flagLsp)
			lang = l
			openfile, opentime := checkRepoPath(repoPath, l)
			if lang == lsp.Golang {
				return
			}
			// Initialize the LSP client
			log.Info("start initialize LSP server %s...\n", lspPath)
			client, err = lsp.NewLSPClient(repoPath, openfile, opentime, lsp.ClientOptions{
				Server:   lspPath,
				Language: l,
				Verbose:  flagVerbose || flagDebug,
			})
			if err != nil {
				log.Error("failed to initialize LSP server: %v\n", err)
				os.Exit(2)
			}
			log.Info("end initialize LSP server")
		},

		Run: func(cmd *cobra.Command, args []string) {
			action := args[0]
			log.Info("start %s repository %s...\n", action, repoPath)
			// Perform the action
			ctx := context.Background()
			switch action {
			case "collect":
				opts := collect.CollectOption{
					LoadExternalSymbol: loadExternalSymbol,
					Excludes:           excludes,
					Language:           lang,
					NoNeedComment:      NoNeedComment,
				}
				repo, err := collectSymbol(ctx, client, repoPath, opts)
				if err != nil {
					log.Error("Failed to collect symbols: %v\n", err)
					os.Exit(3)
				}
				log.Info("all symbols collected, start writing to stdout...\n")
				out, err := json.Marshal(repo)
				if err != nil {
					log.Error("Failed to marshal repository: %v\n", err)
					return
				}
				for n := 0; n < len(out); {
					i, err := os.Stdout.Write(out)
					if err != nil {
						log.Error("Failed to write to stdout: %v\n", err)
						return
					}
					n += i
				}
				return
			default:
				log.Error("Unsupported action: %s\n", action)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&flagLsp, "lsp", "", "Specify the language server path.")
	rootCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose mode.")
	rootCmd.Flags().BoolVarP(&flagDebug, "debug", "d", false, "Debug mode.")
	rootCmd.Flags().BoolVarP(&loadExternalSymbol, "load-external-symbol", "", false, "load external symbols into results")
	rootCmd.Flags().StringSlice("exclude", excludes, "exclude files or directories")
	rootCmd.Flags().BoolVarP(&NoNeedComment, "no-need-comment", "", false, "do not need comment (only works for Go now)")

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		log.Error("Failed to execute command: %v\n", err)
		os.Exit(1)
	}
}

func checkRepoPath(repoPath string, language lsp.Language) (openfile string, wait time.Duration) {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		log.Error("Repository not found: %s\n", repoPath)
		os.Exit(1)
	}
	switch language {
	case lsp.Rust:
		// NOTICE: open the Cargo.toml file is required for Rust projects
		openfile, wait = rust.CheckRepo(repoPath)
	default:
		openfile = ""
		wait = 0
	}

	log.Info("open file '%s' and wait for %d seconds for initialize workspace\n", openfile, wait/time.Second)
	return
}

func checkVerbose(verbose bool, debug bool) {
	if debug {
		log.SetLogLevel(log.DebugLevel)
	} else if verbose {
		log.SetLogLevel(log.InfoLevel)
	} else {
		log.SetLogLevel(log.ErrorLevel)
	}
}

func checkLSP(language string, lspPath string) (l lsp.Language, s string) {
	switch language {
	case "rust":
		l, s = rust.GetDefaultLSP()
	case "golang", "go":
		l = lsp.Golang
		s = ""
		if _, err := exec.LookPath("go"); err != nil {
			if _, err := os.Stat(lspPath); os.IsNotExist(err) {
				log.Error("Go compiler not found, please make it excutable!\n", lspPath)
				os.Exit(1)
			}
		}
		return
	default:
		log.Error("Unsupported language: %s\n", language)
		os.Exit(1)
	}
	// check if lsp excutable
	if lspPath != "" {
		if _, err := exec.LookPath(lspPath); err != nil {
			if _, err := os.Stat(lspPath); os.IsNotExist(err) {
				log.Error("Language server %s not found, please make it excutable!\n", lspPath)
				os.Exit(1)
			}
		}
		s = lspPath
	}

	return
}

func collectSymbol(ctx context.Context, cli *lsp.LSPClient, repoPath string, opts collect.CollectOption) (*uniast.Repository, error) {
	if opts.Language == lsp.Golang {
		return callGoParser(ctx, repoPath, opts)
	}

	collector := collect.NewCollector(repoPath, cli)
	collector.CollectOption = opts
	log.Info("start collecting symbols...\n")
	err := collector.Collect(ctx)
	if err != nil {
		return nil, err
	}
	log.Info("all symbols collected.\n")
	log.Info("start exporting symbols...\n")
	repo, err := collector.Export(ctx)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
