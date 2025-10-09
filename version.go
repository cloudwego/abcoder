/**
 * Copyright 2025 CloudWeGo Authors
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
	"debug/buildinfo"
	"fmt"
	"os"
)

var Version = "0.0.0"

func init() {
	path, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to get executable path: %v\n", err)
		return
	}
	data, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to read executable file: %v\n", err)
		return
	}
	info, err := buildinfo.Read(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to read build info: %v\n", err)
		return
	}
	Version = info.Main.Version
}
