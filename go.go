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

package main

import (
	"context"

	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/golang/parser"
	"github.com/cloudwego/abcoder/lang/uniast"
)

func callGoParser(ctx context.Context, repoPath string, opts collect.CollectOption) (*uniast.Repository, error) {
	goopts := parser.Options{}
	if opts.LoadExternalSymbol {
		goopts.ReferCodeDepth = 1
	}
	if !opts.NoNeedComment {
		goopts.CollectComment = true
	}
	if opts.NeedTest {
		goopts.NeedTest = true
	}
	goopts.Excludes = opts.Excludes
	p := parser.NewParser(repoPath, repoPath, goopts)
	repo, err := p.ParseRepo()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}
