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

package main

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) < 2 {
		println("missing argument")
		os.Exit(1)
	}
	// content, err := readFile(os.Args[1])
	// if err != nil {
	// 	println(err.Error())
	// 	os.Exit(1)
	// }
	content := []byte("{}")
	InternalFunc(content)
	var s = new(Struct)
	s.Field4 = InternalFunc
	s.InternalMethod(content)
}

func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	return ioutil.ReadAll(file)
}
