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

package parser

import (
	"fmt"
	"os"
	"regexp"
)

type Options struct {
	ReferCodeDepth int
	Excludes       []string
	CollectComment bool
	NeedTest       bool
}

// type Option func(options *Options)

// func WithReferCodeDepth(depth int) Option {
// 	return func(options *Options) {
// 		options.ReferCodeDepth = depth
// 	}
// }

func compileExcludes(excludes []string) (ret []*regexp.Regexp) {
	for _, ex := range excludes {
		r, e := regexp.Compile(ex)
		if e != nil {
			fmt.Fprintf(os.Stderr, "compile exlude-path regexp failed: %s", ex)
			continue
		}
		ret = append(ret, r)
	}
	return ret
}

// func WithCollectComment(collect bool) Option {
// 	return func(options *Options) {
// 		options.CollectComment = collect
// 	}
// }
