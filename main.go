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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/agent"
	"github.com/cloudwego/abcoder/llm/mcp"
	"github.com/cloudwego/abcoder/llm/tool"
)

const Usage = `abcoder <Action> [Language] <Path> [Flags]
Action:
   parse        parse the specific repo and write its UniAST (to stdout by default)
   write        write the specific UniAST back to codes
   mcp          run as a MCP server for all repo ASTs (*.json) in the specific directory
   agent        run as an Agent for all repo ASTs (*.json) in the specific directory. WIP: only support code-analyzing at present.
   version      print the version of abcoder
Language:
   rust         for rust codes
   cxx          for c codes (cpp support is on the way)
   go           for golang codes
   python       for python codes
`

func main() {
	flags := flag.NewFlagSet("abcoder", flag.ExitOnError)

	flagHelp := flags.Bool("h", false, "Show help message.")
	flagVerbose := flags.Bool("verbose", false, "Verbose mode.")
	flagOutput := flags.String("o", "", "Output path.")
	flagLsp := flags.String("lsp", "", "Specify the language server path.")
	flagProfile := flags.String("profile", "", "Specify the CPU profile data path. Leave empty to disable profiling.")

	var opts lang.ParseOptions
	flags.BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "load external symbols into results")
	flags.BoolVar(&opts.NoNeedComment, "no-need-comment", false, "not need comment (only works for Go now)")
	flags.BoolVar(&opts.NotNeedTest, "no-need-test", false, "not need parse test files (only works for Go now)")
	flags.BoolVar(&opts.LoadByPackages, "load-by-packages", false, "load by packages (only works for Go now)")
	flags.Var((*StringArray)(&opts.Excludes), "exclude", "exclude files or directories, support multiple values")
	flags.Var((*StringArray)(&opts.Includes), "include", "include files or directories, support multiple values")
	flags.StringVar(&opts.LSPCachePath, "lsp-cache-path", "", "the path used for caching LSP requests (set to empty to disable saving cache to disk)")
	flags.StringVar(&opts.RepoID, "repo-id", "", "specify the repo id")
	flags.StringVar(&opts.TSConfig, "tsconfig", "", "tsconfig path (only works for TS now)")
	flags.Var((*StringArray)(&opts.TSSrcDir), "ts-src-dir", "src-dir path (only works for TS now)")

	var wopts lang.WriteOptions
	flags.StringVar(&wopts.Compiler, "compiler", "", "destination compiler path.")

	var aopts agent.AgentOptions
	flags.IntVar(&aopts.MaxSteps, "agent-max-steps", 50, "specify the max steps that the agent can run for each time")
	flags.IntVar(&aopts.MaxHistories, "agent-max-histories", 10, "specify the max histories that the agent can use")

	flags.Usage = func() {
		fmt.Fprint(os.Stderr, Usage)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flags.Usage()
		os.Exit(1)
	}

	action := strings.ToLower(os.Args[1])

	switch action {
	case "version":
		fmt.Fprintf(os.Stdout, "%s\n", Version)

	case "parse":
		language, uri := parseArgsAndFlags(flags, true, flagHelp, flagVerbose)

		// Profiling with Ctrl-C support
		if flagProfile != nil && *flagProfile != "" {
			f, _ := os.Create(*flagProfile)
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigs
				pprof.StopCPUProfile()
				f.Close()
				os.Exit(0)
			}()
		}

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
			opts.Verbose = true
		}

		opts.Language = language

		if language == uniast.TypeScript {
			if err := parseTSProject(context.Background(), uri, opts, flagOutput); err != nil {
				log.Error("Failed to parse: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if flagLsp != nil {
			opts.LSP = *flagLsp
		}

		out, err := lang.Parse(context.Background(), uri, opts)
		if err != nil {
			log.Error("Failed to parse: %v\n", err)
			os.Exit(1)
		}

		if flagOutput != nil && *flagOutput != "" {
			if err := utils.MustWriteFile(*flagOutput, out); err != nil {
				log.Error("Failed to write output: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", out)
		}

	case "write":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Arguement Path is required\n")
			os.Exit(1)
		}

		repo, err := uniast.LoadRepo(uri)
		if err != nil {
			log.Error("Failed to load repo: %v\n", err)
			os.Exit(1)
		}

		if flagOutput != nil && *flagOutput != "" {
			wopts.OutputDir = *flagOutput
		} else {
			wopts.OutputDir = filepath.Base(repo.Path)
		}

		if err := lang.Write(context.Background(), repo, wopts); err != nil {
			log.Error("Failed to write: %v\n", err)
			os.Exit(1)
		}

	case "mcp":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Arguement Path is required\n")
			os.Exit(1)
		}

		svr := mcp.NewServer(mcp.ServerOptions{
			ServerName:    "abcoder",
			ServerVersion: Version,
			Verbose:       *flagVerbose,
			ASTReadToolsOptions: tool.ASTReadToolsOptions{
				RepoASTsDir: uri,
			},
		})
		if err := svr.ServeStdio(); err != nil {
			log.Error("Failed to run MCP server: %v\n", err)
			os.Exit(1)
		}

	case "agent":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Arguement Path is required\n")
			os.Exit(1)
		}

		aopts.ASTsDir = uri
		aopts.Model.APIType = llm.NewModelType(os.Getenv("API_TYPE"))
		if aopts.Model.APIType == llm.ModelTypeUnknown {
			log.Error("env API_TYPE is required")
			os.Exit(1)
		}
		aopts.Model.APIKey = os.Getenv("API_KEY")
		if aopts.Model.APIKey == "" {
			log.Error("env API_KEY is required")
			os.Exit(1)
		}
		aopts.Model.ModelName = os.Getenv("MODEL_NAME")
		if aopts.Model.ModelName == "" {
			log.Error("env MODEL_NAME is required")
			os.Exit(1)
		}
		aopts.Model.BaseURL = os.Getenv("BASE_URL")

		ag := agent.NewAgent(aopts)
		ag.Run(context.Background())

	}
}

func parseArgsAndFlags(flags *flag.FlagSet, needLang bool, flagHelp *bool, flagVerbose *bool) (language uniast.Language, uri string) {
	if len(os.Args) < 3 {
		flags.Usage()
		os.Exit(1)
	}

	if needLang {
		language = uniast.NewLanguage(os.Args[2])
		if language == uniast.Unknown {
			fmt.Fprintf(os.Stderr, "unsupported language: %s\n", os.Args[2])
			os.Exit(1)
		}
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "arguement Path is required\n")
			os.Exit(1)
		}
		uri = os.Args[3]
		if len(os.Args) > 4 {
			flags.Parse(os.Args[4:])
		}
	} else {
		uri = os.Args[2]
		if len(os.Args) > 3 {
			flags.Parse(os.Args[3:])
		}
	}

	if flagHelp != nil && *flagHelp {
		flags.Usage()
		os.Exit(0)
	}

	if flagVerbose != nil && *flagVerbose {
		log.SetLogLevel(log.DebugLevel)
	}

	return language, uri
}

type StringArray []string

func (s *StringArray) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *StringArray) String() string {
	return strings.Join(*s, ",")
}

func parseTSProject(ctx context.Context, repoPath string, opts lang.ParseOptions, outputFlag *string) error {
	if outputFlag == nil {
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
	if *outputFlag != "" {
		args = append(args, "--output", *outputFlag)
	}

	cmd := exec.CommandContext(ctx, parserPath, args...)
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=65536")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info("Running abcoder-ts-parser with args: %v", args)

	return cmd.Run()
}
