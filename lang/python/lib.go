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

package python

import (
	"time"

	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

const MaxWaitDuration = 5 * time.Second

func GetDefaultLSP() (lang uniast.Language, name string) {
	// Use custom PyLSP.
	return uniast.Python, "pylsp"
}

func CheckRepo(repo string) (string, time.Duration) {
	openfile := ""
	// TODO: check if the project compiles.

	// NOTICE: wait for Rust projects based on code files
	_, size := utils.CountFiles(repo, ".py", "SKIPDIR")
	wait := 2*time.Second + time.Second*time.Duration(size/1024)
	if wait > MaxWaitDuration {
		wait = MaxWaitDuration
	}
	return openfile, wait
}
