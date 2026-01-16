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
	"github.com/spf13/cobra"
)

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
		Long: `ABCoder is a universal code analysis tool that converts source code to UniAST format.

It supports multiple programming languages and provides various subcommands for parsing,
writing, and analyzing code structures.`,
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
		Short: "Print abcoder version information",
		Long: `Output the version number and build metadata in the format: vX.X.Y-BUILD.

Use this command to verify installation or when reporting issues.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "%s\n", version.Version)
		},
	}
}

func newParseCmd() *cobra.Command {
	var (
		flagOutput string
		flagLsp    string
		javaHome   string
		opts       lang.ParseOptions
	)

	cmd := &cobra.Command{
		Use:   "parse <language> <path>",
		Short: "Parse repository and export to UniAST JSON format",
		Long: `Parse the specified repository and generate its Universal AST representation.

By default, outputs to stdout. Use --output to write to a file.

Language Support:
  go      - Go projects
  rust    - Rust projects
  cxx      - C/C++ projects
  python   - Python projects
  ts       - TypeScript projects
  js       - JavaScript projects
  java     - Java projects`,
		Example: `abcoder parse go ./my-project -o ast.json`,
		Args:    cobra.ExactArgs(2),
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
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output path for UniAST JSON (default: stdout).")
	cmd.Flags().StringVar(&flagLsp, "lsp", "", "Path to Language Server Protocol executable. Required for languages with LSP support (e.g., Java).")
	cmd.Flags().StringVar(&javaHome, "java-home", "", "Java installation directory (JAVA_HOME). Required when using LSP for Java.")
	cmd.Flags().BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "Load external symbol references into AST results (slower but more complete).")
	cmd.Flags().BoolVar(&opts.NoNeedComment, "no-need-comment", false, "Skip parsing code comments (only works for Go).")
	cmd.Flags().BoolVar(&opts.NotNeedTest, "no-need-test", false, "Skip test files during parsing (only works for Go).")
	cmd.Flags().BoolVar(&opts.LoadByPackages, "load-by-packages", false, "Load packages one by one instead of all at once (only works for Go, uses more memory).")
	cmd.Flags().StringSliceVar(&opts.Excludes, "exclude", []string{}, "Files or directories to exclude from parsing (can be specified multiple times).")
	cmd.Flags().StringVar(&opts.RepoID, "repo-id", "", "Custom identifier for this repository (useful for multi-repo scenarios).")
	cmd.Flags().StringVar(&opts.TSConfig, "tsconfig", "", "Path to tsconfig.json file for TypeScript project configuration.")
	cmd.Flags().StringSliceVar(&opts.TSSrcDir, "ts-src-dir", []string{}, "Additional TypeScript source directories (can be specified multiple times).")

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

	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output directory for generated code files (default: <basename of input file>).")
	cmd.Flags().StringVar(&wopts.Compiler, "compiler", "", "Path to compiler executable (language-specific).")

	return cmd
}

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <directory>",
		Short: "Start MCP server for AST files",
		Long: `Start a Model Context Protocol (MCP) server that provides AST reading tools.

The server communicates via stdio and can be integrated with Claude Code or other MCP clients.

It serves all *.json AST files in the specified directory.`,
		Example: `abcoder mcp ./asts/`,
		Args:    cobra.ExactArgs(1),
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
		Use:   "init-spec [project-path]",
		Short: "Initialize ABCoder integration for Claude Code",
		Long: `Initialize ABCoder integration by copying .claude directory and configuring MCP servers.

This sets up Claude Code to use ABCoder for code analysis.

The path defaults to the current directory if not specified.

The command will:
1. Copy the .claude configuration directory
2. Configure MCP server settings in Claude's config.json`,
		Example: `abcoder init-spec /path/to/project`,
		Args:    cobra.MaximumNArgs(1),
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
		Use:   "agent <directory>",
		Short: "Run AI agent with code analysis capabilities",
		Long: `Start an autonomous AI agent that can perform code analysis tasks using LLM.

The agent reads AST files from the specified directory and can perform various
code analysis operations.

Required Environment Variables:
  API_TYPE   LLM provider type (e.g., openai, anthropic)
  API_KEY    LLM API authentication key
  MODEL_NAME Model identifier (e.g., gpt-4, claude-3-opus-20240229)
  BASE_URL    (Optional) Custom API base URL

Examples:
  # Basic usage with OpenAI
  API_TYPE=openai API_KEY=sk-xxx MODEL_NAME=gpt-4 \
    abcoder agent ./asts/

  # With custom API endpoint and step limit
  API_TYPE=custom API_KEY=xxx MODEL_NAME=my-model BASE_URL=https://api.example.com \
    abcoder agent ./asts/ --agent-max-steps 100`,
		Args: cobra.ExactArgs(1),
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

	cmd.Flags().IntVar(&aopts.MaxSteps, "agent-max-steps", 50, "Maximum number of agent reasoning steps per task (default: 50). Higher values allow more complex tasks but increase cost.")
	cmd.Flags().IntVar(&aopts.MaxHistories, "agent-max-histories", 10, "Maximum number of conversation histories to maintain for context (default: 10).")

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
