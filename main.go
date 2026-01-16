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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	interutils "github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/agent"
	"github.com/cloudwego/abcoder/llm/mcp"
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/cloudwego/abcoder/version"
)

const Usage = `abcoder <Action> [Language] <Path> [Flags]
Action:
   parse        parse the specific repo and write its UniAST (to stdout by default)
   write        write the specific UniAST back to codes
   mcp          run as a MCP server for all repo ASTs (*.json) in the specific directory
   agent        run as an Agent for all repo ASTs (*.json) in the specific directory. WIP: only support code-analyzing at present.
   init-spec    initialize ABCoder integration for Claude Code (copies .claude directory and configures MCP servers)
   version      print the version of abcoder
Language:
   go           for golang codes
   rust         for rust codes
   cxx          for c codes (cpp support is on the way)
   python       for python codes
   ts           for typescript codes
   js           for javascript codes
   java         for java codes
`

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "abcoder",
		Short: "Universal AST parser and writer for multi-language",
		Long:  Usage,
	}

	// Global flags
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode.")

	// Add subcommands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newParseCmd())
	cmd.AddCommand(newWriteCmd())
	cmd.AddCommand(newMcpCmd())
	cmd.AddCommand(newInitSpecCmd())
	cmd.AddCommand(newAgentCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version of abcoder",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "%s\n", version.Version)
		},
	}
}

func newParseCmd() *cobra.Command {
	var (
		flagOutput    string
		flagLsp      string
		javaHome     string
		opts         lang.ParseOptions
	)

	cmd := &cobra.Command{
		Use:   "parse <language> <path>",
		Short: "parse the specific repo and write its UniAST (to stdout by default)",
		Args:  cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate language
			language := uniast.NewLanguage(args[0])
			if language == uniast.Unknown {
				return fmt.Errorf("unsupported language: %s", args[0])
			}
			opts.Language = language
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				log.SetLogLevel(log.DebugLevel)
				opts.Verbose = true
			}

			language := uniast.NewLanguage(args[0])
			uri := args[1]

			if language == uniast.TypeScript {
				if err := parseTSProject(context.Background(), uri, opts, flagOutput); err != nil {
					log.Error("Failed to parse: %v\n", err)
					return err
				}
				return nil
			}

			if flagLsp != "" {
				opts.LSP = flagLsp
			}

			lspOptions := make(map[string]string)
			if javaHome != "" {
				lspOptions["java.home"] = javaHome
			}
			opts.LspOptions = lspOptions

			out, err := lang.Parse(context.Background(), uri, opts)
			if err != nil {
				log.Error("Failed to parse: %v\n", err)
				return err
			}

			if flagOutput != "" {
				if err := utils.MustWriteFile(flagOutput, out); err != nil {
					log.Error("Failed to write output: %v\n", err)
					return err
				}
			} else {
				fmt.Fprintf(os.Stdout, "%s\n", out)
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output path.")
	cmd.Flags().StringVar(&flagLsp, "lsp", "", "Specify the language server path.")
	cmd.Flags().StringVar(&javaHome, "java-home", "", "java home")
	cmd.Flags().BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "load external symbols into results")
	cmd.Flags().BoolVar(&opts.NoNeedComment, "no-need-comment", false, "not need comment (only works for Go now)")
	cmd.Flags().BoolVar(&opts.NotNeedTest, "no-need-test", false, "not need parse test files (only works for Go now)")
	cmd.Flags().BoolVar(&opts.LoadByPackages, "load-by-packages", false, "load by packages (only works for Go now)")
	cmd.Flags().StringSliceVar(&opts.Excludes, "exclude", []string{}, "exclude files or directories, support multiple values")
	cmd.Flags().StringVar(&opts.RepoID, "repo-id", "", "specify the repo id")
	cmd.Flags().StringVar(&opts.TSConfig, "tsconfig", "", "tsconfig path (only works for TS now)")
	cmd.Flags().StringSliceVar(&opts.TSSrcDir, "ts-src-dir", []string{}, "src-dir path (only works for TS now)")

	return cmd
}

func newWriteCmd() *cobra.Command {
	var (
		flagOutput string
		wopts      lang.WriteOptions
	)

	cmd := &cobra.Command{
		Use:   "write <path>",
		Short: "write the specific UniAST back to codes",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("argument Path is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				log.SetLogLevel(log.DebugLevel)
			}

			uri := args[0]

			repo, err := uniast.LoadRepo(uri)
			if err != nil {
				log.Error("Failed to load repo: %v\n", err)
				return err
			}

			if flagOutput != "" {
				wopts.OutputDir = flagOutput
			} else {
				wopts.OutputDir = filepath.Base(repo.Path)
			}

			if err := lang.Write(context.Background(), repo, wopts); err != nil {
				log.Error("Failed to write: %v\n", err)
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output path.")
	cmd.Flags().StringVar(&wopts.Compiler, "compiler", "", "destination compiler path.")

	return cmd
}

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <path>",
		Short: "run as a MCP server for all repo ASTs (*.json) in the specific directory",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("argument Path is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")

			uri := args[0]

			svr := mcp.NewServer(mcp.ServerOptions{
				ServerName:    "abcoder",
				ServerVersion: version.Version,
				Verbose:       verbose,
				ASTReadToolsOptions: tool.ASTReadToolsOptions{
					RepoASTsDir: uri,
				},
			})
			if err := svr.ServeStdio(); err != nil {
				log.Error("Failed to run MCP server: %v\n", err)
				return err
			}

			return nil
		},
	}
}

func newInitSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-spec [path]",
		Short: "initialize ABCoder integration for Claude Code (copies .claude directory and configures MCP servers)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				log.SetLogLevel(log.DebugLevel)
			}

			var uri string
			if len(args) > 0 {
				uri = args[0]
			}

			if err := interutils.RunInitSpec(uri); err != nil {
				log.Error("Failed to init-spec: %v\n", err)
				return err
			}

			return nil
		},
	}
}

func newAgentCmd() *cobra.Command {
	var (
		aopts agent.AgentOptions
	)

	cmd := &cobra.Command{
		Use:   "agent <path>",
		Short: "run as an Agent for all repo ASTs (*.json) in the specific directory. WIP: only support code-analyzing at present.",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("argument Path is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				log.SetLogLevel(log.DebugLevel)
			}

			uri := args[0]

			aopts.ASTsDir = uri
			aopts.Model.APIType = llm.NewModelType(os.Getenv("API_TYPE"))
			if aopts.Model.APIType == llm.ModelTypeUnknown {
				log.Error("env API_TYPE is required")
				return fmt.Errorf("env API_TYPE is required")
			}
			aopts.Model.APIKey = os.Getenv("API_KEY")
			if aopts.Model.APIKey == "" {
				log.Error("env API_KEY is required")
				return fmt.Errorf("env API_KEY is required")
			}
			aopts.Model.ModelName = os.Getenv("MODEL_NAME")
			if aopts.Model.ModelName == "" {
				log.Error("env MODEL_NAME is required")
				return fmt.Errorf("env MODEL_NAME is required")
			}
			aopts.Model.BaseURL = os.Getenv("BASE_URL")

			ag := agent.NewAgent(aopts)
			ag.Run(context.Background())

			return nil
		},
	}

	cmd.Flags().IntVar(&aopts.MaxSteps, "agent-max-steps", 50, "specify the max steps that the agent can run for each time")
	cmd.Flags().IntVar(&aopts.MaxHistories, "agent-max-histories", 10, "specify the max histories that the agent can use")

	return cmd
}

func parseTSProject(ctx context.Context, repoPath string, opts lang.ParseOptions, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	parserPath, err := exec.LookPath("abcoder-ts-parser")
	if err != nil {
		log.Info("abcoder-ts-parser not found, installing...")
		cmd := exec.Command("npm", "install", "-g", "abcoder-ts-parser")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install abcoder-ts-parser: %v", err)
		}
		parserPath, err = exec.LookPath("abcoder-ts-parser")
		if err != nil {
			return fmt.Errorf("failed to find abcoder-ts-parser after installation: %v", err)
		}
	}

	args := []string{"parse", repoPath}
	if len(opts.TSSrcDir) > 0 {
		args = append(args, "--src", strings.Join(opts.TSSrcDir, ","))
	}
	if opts.TSConfig != "" {
		args = append(args, "--tsconfig", opts.TSConfig)
	}
	if outputPath != "" {
		args = append(args, "--output", outputPath)
	}

	cmd := exec.CommandContext(ctx, parserPath, args...)
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=65536")
	cmd.Env = append(cmd.Env, "ABCODER_TOOL_VERSION="+version.Version)
	cmd.Env = append(cmd.Env, "ABCODER_AST_VERSION="+uniast.Version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info("Running abcoder-ts-parser with args: %v", args)

	return cmd.Run()
}
