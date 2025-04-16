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
	"strings"

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/lsp"
)

const Usage = `abcoder <Action> <Language> <RepoPath> [Flags]
Action:
   parse		Parse the whole repo and export AST
Language:
   rust			For rust codes
   go  			For go codes
RepoPath:
   The directory path of the repo to parse
`

func main() {
	flags := flag.NewFlagSet("abcoder", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, Usage)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, Usage)
		os.Exit(1)
	}

	action := os.Args[1]
	language := lsp.NewLanguage(os.Args[2])
	if language == lsp.Unknown {
		fmt.Fprintf(os.Stderr, "unsupported language: %s\n", os.Args[2])
		os.Exit(1)
	}
	repoPath := os.Args[3]

	var flagLsp string
	var flagVerbose bool
	flags.StringVar(&flagLsp, "lsp", "", "Specify the language server path.")
	flags.BoolVar(&flagVerbose, "verbose", false, "Verbose mode.")

	switch action {
	case "parse":
		var opts lang.ParseOptions
		flags.BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "load external symbols into results")
		flags.BoolVar(&opts.NoNeedComment, "no-need-comment", false, "do not need comment (only works for Go now)")
		flags.BoolVar(&opts.NeedTest, "need-test", false, "need parse test files (only works for Go now)")
		flags.Var((*StringArray)(&opts.Excludes), "exclude", "exclude files or directories, support multiple values")

		flags.Parse(os.Args[4:])
		if flagVerbose {
			log.SetLogLevel(log.DebugLevel)
		}

		opts.Language = language
		opts.LSP = flagLsp
		opts.Verbose = flagVerbose

		out, err := lang.Parse(context.Background(), repoPath, opts)
		if err != nil {
			log.Error("Failed to parse: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "%s\n", out)
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
