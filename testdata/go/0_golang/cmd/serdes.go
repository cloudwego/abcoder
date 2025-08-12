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
	"io"

	"a.b/c/pkg"
	"a.b/c/pkg/entity"
	"github.com/bytedance/sonic"
)

func InternalFunc(in []byte) {
	var s = new(pkg.CaseStruct)
	if err := pkg.Case_Func(in, s); err != nil {
		println(err.Error())
	}
	var x = new(entity.MyStruct)
	if v := s.CaseMethod(in, x); v != 0 {
		println(v)
	}
}

func DuplicateName() {
}

type Struct struct {
	Field1 string
	Field2 pkg.CaseStruct
	Field3 sonic.Config
	Field4 func(in []byte)
}

func (s *Struct) InternalMethod(in []byte) {
	if err := sonic.Unmarshal(in, &s); err != nil {
		println(err.Error())
	}
	if err := s.Field2.CaseMethod(in, nil); err != 0 {
		println(err)
	}
}

func (s *Struct) DuplicateName(in []byte) {
}

var VarString *string

var VarInt = 1

var VarSlice = []int{1, 2, 3}

var VarFunc io.Reader

var VarpkgStruct pkg.CaseStruct

var VarStruct *Struct

var Var1, Var2 = 1, ""

const Con1, Con2 = 1, ""
