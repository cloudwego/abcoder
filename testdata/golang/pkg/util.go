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

package pkg

import (
	"encoding/json"
	"flag"

	"a.b/c/pkg/entity"
	"github.com/bytedance/sonic"
)

func Case_Func(in []byte, s *CaseStruct) error {
	return sonic.Unmarshal(in, s)
}

type CaseStruct struct {
	FieldPremitive         string
	FieldType              Integer
	FieldExternalType      entity.MyStruct
	FieldInterface         InterfaceA
	FieldExternalInterface entity.InterfaceB
	FieldClosuer           func(in string) int
}

type InterfaceA interface {
	String() string
}

func (s *CaseStruct) CaseMethod(in []byte, x *entity.MyStruct) Integer {
	_ = json.Unmarshal(in, x)
	return Integer(0)
}

func (s *CaseStruct) String() string {
	return s.FieldExternalType.String()
}

var GlobalVar int

func Case_Func_GloabVar() int {
	return GlobalVar + entity.G1
}

type Integer int

func Case_Func_RefType() int {
	var x entity.Integer
	var y Integer
	return int(x) + int(y)
}

func Case_Func_Method() {
	s := &CaseStruct{}
	_ = s.CaseMethod(nil, nil)
	s2 := &entity.MyStruct{}
	_ = s2.String()
}

func Case_Func_Func() {
	_ = Case_Func(nil, nil)
	_ = entity.A("")
}

func Case_All(in string, a entity.MyStruct, b CaseStruct) (int, Integer, entity.Integer) {
	var x entity.Integer
	var y Integer
	_ = int(x) + int(y)
	var s CaseStruct
	_ = s.CaseMethod(nil, nil)
	var a2 entity.MyStruct
	_ = a2.String()
	b.FieldClosuer = func(in string) int { return 0 }
	_ = b.CaseMethod(nil, nil)
	a.MyStructD = entity.MyStructD{}
	_ = a.String()
	_ = Case_Func(nil, nil)
	_ = entity.A("")
	_ = GlobalVar + entity.G1
	return 0, 0, 0
}

const (
	Enum1               = 1
	Enum2, Enum3        = 2, 3
	Enum4        string = "4"
)

var (
	Var1              = 1
	Var2, Var3        = 2, 3
	Var4       string = "4"
	Var5              = []string{"a"}
	Var6              = func() {}
	Var7              = flag.Bool("flag", false, "usage")
	Var8              = sonic.Config{}
)

func Case_Func_Global() {
	_ = Enum1
	_ = Enum2
	_ = Enum3
	_ = Enum4
	_ = Var1
	_ = Var2
	_ = Var3
	_ = Var4
	_ = Var5
	_ = Var6
	_ = Var7
	_ = Var8
	_ = entity.G1
	_ = entity.V1
}

// Type is Result type
type Type int

const (
	// Null is a null json value
	Null Type = iota
	// False is a json false boolean
	False
	// Number is json number
	Number
	// String is a json string
	String
	// True is a json true boolean
	True
	// JSON is a raw block of JSON
	JSON
)

func CaseStrucLiterMethod() {
	_ = (&CaseStruct{
		FieldPremitive:         "a",
		FieldType:              1,
		FieldExternalType:      entity.MyStruct{},
		FieldInterface:         nil,
		FieldExternalInterface: nil,
		FieldClosuer:           nil,
	}).CaseMethod(nil, nil)
}
