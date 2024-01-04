/**
 * Copyright 2024 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"a.b/c/pkg"
	"github.com/bytedance/sonic"
)

func InternalFunc(in []byte) {
	var s = new(pkg.StructA)
	if err := pkg.FuncA(in, s); err != nil {
		println(err.Error())
	}
	if err := s.MethodA(in); err != nil {
		println(err.Error())
	}
}

type Struct struct {
	Field1 string
	Field2 pkg.StructA
	Field3 sonic.Config
}

func (s *Struct) InternalMethod(in []byte) {
	if err := sonic.Unmarshal(in, &s); err != nil {
		println(err.Error())
	}
	if err := s.Field2.MethodA(in); err != nil {
		println(err.Error())
	}
}
