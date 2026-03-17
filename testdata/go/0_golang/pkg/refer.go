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
	"a.b/c/pkg/entity"
	"github.com/bytedance/sonic"
)

type Case_Refer_Type sonic.Config

func Case_Refer_Func() {
	sonic.Get(nil, "")
	sonic.Pretouch(nil)
}

func Case_Refer_Method() {
	n := sonic.Config{}
	n.Froze()
}

var Case_Refer_Var = sonic.ConfigStd

type Obj struct{}

func (o Obj) Call() Obj {
	return o
}

func (o Obj) CallFunc(f func(Obj) Obj) Obj {
	return f(o)
}

func Case_Chain_Selector() {
	var obj Obj
	obj.Call().Call()
}

func Case_Closure() {
	var obj Obj
	obj.CallFunc(func(o Obj) Obj {
		return o.Call()
	})
}

type Case_Annoy_Struct struct {
	A struct {
		B int
	}
	C int
}

const Case_Ref_Const = entity.G1
