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
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

const Usage = `abcoder <Action> <Language> <URI> [Flags]
Action:
   parse		parse the whole repo and output UniAST
   write        write the UniAST to the output directory
Language:
   rust			for rust codes
   cxx  		for c codes (cpp support is on the way)
   go  			for golang codes
`

func main() {
	flags := flag.NewFlagSet("abcoder", flag.ExitOnError)

	flagHelp := flags.Bool("h", false, "Show help message.")

	flagVerbose := flags.Bool("verbose", false, "Verbose mode.")

	flagOutput := flags.String("o", "", "Output path.")

	var opts lang.ParseOptions
	flags.BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "load external symbols into results")
	flags.BoolVar(&opts.NoNeedComment, "no-need-comment", false, "do not need comment (only works for Go now)")
	flags.BoolVar(&opts.NeedTest, "need-test", false, "need parse test files (only works for Go now)")
	flags.BoolVar(&opts.LoadByPackages, "load-by-packages", false, "load by packages (only works for Go now)")
	flags.Var((*StringArray)(&opts.Excludes), "exclude", "exclude files or directories, support multiple values")
	flags.StringVar(&opts.RepoID, "repo-id", "", "specify the repo id")
	flagLsp := flags.String("lsp", "", "Specify the language server path.")

	var wopts lang.WriteOptions
	flags.StringVar(&wopts.Compiler, "compiler", "", "destination compiler path.")

	flags.Usage = func() {
		fmt.Fprint(os.Stderr, Usage)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if len(os.Args) < 4 {
		// call flags.Usage()
		flags.Usage()
		os.Exit(1)
	}
	action := strings.ToLower(os.Args[1])
	language := uniast.NewLanguage(os.Args[2])
	if language == uniast.Unknown {
		fmt.Fprintf(os.Stderr, "unsupported language: %s\n", os.Args[2])
		os.Exit(1)
	}
	uri := os.Args[3]

	flags.Parse(os.Args[4:])
	if flagHelp != nil && *flagHelp {
		flags.Usage()
		os.Exit(0)
	}

	switch action {
	case "parse":

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
			opts.Verbose = true
		}

		opts.Language = language
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
		repo, err := uniast.LoadRepo(uri)
		if err != nil {
			log.Error("Failed to load repo: %v\n", err)
			os.Exit(1)
		}

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
		}
		if flagOutput != nil && *flagOutput != "" {
			wopts.OutputDir = *flagOutput
		} else {
			wopts.OutputDir = filepath.Base(repo.Name)
		}

		if err := lang.Write(context.Background(), repo, wopts); err != nil {
			log.Error("Failed to write: %v\n", err)
			os.Exit(1)
		}
	}
}

type StringArray []string

func (s *StringArray) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *StringArray) String() string {
	return strings.Join(*s, ",")
}
