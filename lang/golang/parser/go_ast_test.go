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
	"bytes"
	"encoding/json"
	"testing"

	. "github.com/cloudwego/abcoder/lang/uniast"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func Test_goParser_GeMainOnDepends(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
		opts        Options
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test",
			fields: fields{
				homePageDir: "../../../../../tmp/cloudwego/kitex",
				opts: Options{
					ReferCodeDepth: 1,
					CollectComment: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir, tt.fields.opts)
			n, err := p.getNode(NewIdentity("github.com/cloudwego/kitex", "github.com/cloudwego/kitex/pkg/generic", "ParseContent"))
			if err != nil {
				t.Fatal(err)
			}
			if n == nil {
				t.Fatal("nil get node")
			}
			spew.Dump(p)
			pj, err := json.MarshalIndent(n, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			println(string(pj))
			ids, err := p.searchName("main")
			if err != nil {
				t.Log(err.Error())
			}
			if len(ids) == 0 {
				t.Fatal("not found")
			}
			spew.Dump(ids)
			dep, e := p.getNode(Identity{"github.com/cloudwego/kitex", "github.com/cloudwego/kitex/pkg/generic", "BinaryThriftGeneric"})
			if e != nil {
				t.Fatal(e)
			}
			spew.Dump(dep.(*Function).Content)
			var repo = NewRepository(tt.fields.modName)
			for _, id := range ids {
				loadNode(p, id.PkgPath, id.Name, &repo)
			}
			spew.Dump(repo)
		})
	}
}

func TestCases(t *testing.T) {
	type fields struct {
		mod  string
		pkg  string
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		refered *Identity
		want    string
	}{
		{
			name:   "func + third-party",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Func"},
			want:   `{"Exported":true,"IsMethod":false,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Case_Func","File":"util.go","Line":27,"Content":"func Case_Func(in []byte, s *CaseStruct) error {\n\treturn sonic.Unmarshal(in, s)\n}","Params":{"CaseStruct":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"CaseStruct"}},"FunctionCalls":{"sonic.Unmarshal":{"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic","Name":"Unmarshal"}}}`,
		},
		{
			name:   "method + stdlib",
			fields: fields{"a.b/c", "a.b/c/pkg", "CaseStruct.CaseMethod"},
			want:   `{"Exported":true,"IsMethod":true,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"CaseStruct.CaseMethod","File":"util.go","Line":44,"Content":"func (s *CaseStruct) CaseMethod(in []byte, x *entity.MyStruct) Integer {\n\t_ = json.Unmarshal(in, x)\n\treturn Integer(0)\n}","Receiver":{"IsPointer":false,"Type":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"CaseStruct"},"Name":"s"},"Params":{"MyStruct":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg/entity","Name":"MyStruct"}},"Results":{"Integer":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Integer"}},"Types":{"Integer":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Integer"}}}`,
		},
		{
			name:   "struct",
			fields: fields{"a.b/c", "a.b/c/pkg", "CaseStruct"},
			want:   `{"Exported":true,"TypeKind":0,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"CaseStruct","File":"util.go","Line":31,"Content":"type CaseStruct struct {\n\tFieldPremitive         string\n\tFieldType              Integer\n\tFieldExternalType      entity.MyStruct\n\tFieldInterface         InterfaceA\n\tFieldExternalInterface entity.InterfaceB\n\tFieldClosuer           func(in string) int\n}","SubStruct":{"Integer":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Integer"},"InterfaceA":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"InterfaceA"},"InterfaceB":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg/entity","Name":"InterfaceB"},"MyStruct":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg/entity","Name":"MyStruct"}}}`,
		},
		{
			name:   "global vars in a func",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Func_GloabVar"},
			want: `{
				"Exported": true,
				"IsMethod": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Case_Func_GloabVar",
				"File": "util.go",
				"Line": 55,
				"Content": "func Case_Func_GloabVar() int {\n\treturn GlobalVar + entity.G1\n}",
				"GolobalVars": {
				  "GlobalVar": {
					"ModPath": "a.b/c",
					"PkgPath": "a.b/c/pkg",
					"Name": "GlobalVar"
				  },
				  "entity.G1": {
					"ModPath": "a.b/c",
					"PkgPath": "a.b/c/pkg/entity",
					"Name": "G1"
				  }
				}
			  }`,
		},
		{
			name:   "types in a func",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Func_RefType"},
			want: `{
				"Exported": true,
				"IsMethod": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Case_Func_RefType",
				"File": "util.go",
				"Line": 61,
				"Content": "func Case_Func_RefType() int {\n\tvar x entity.Integer\n\tvar y Integer\n\treturn int(x) + int(y)\n}",
				"Types": {
				  "Integer": {
					"ModPath": "a.b/c",
					"PkgPath": "a.b/c/pkg",
					"Name": "Integer"
				  },
				  "entity.Integer": {
					"ModPath": "a.b/c",
					"PkgPath": "a.b/c/pkg/entity",
					"Name": "Integer"
				  }
				}
			  }`,
		},
		{
			name:   "func in a func",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Func_Func"},
			want: `{
				"Exported": true,
				"IsMethod": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Case_Func_Func",
				"File": "util.go",
				"Line": 74,
				"Content": "func Case_Func_Func() {\n\t_ = Case_Func(nil, nil)\n\t_ = entity.A(\"\")\n}",
				"FunctionCalls": {
					"Case_Func": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg",
						"Name": "Case_Func"
					},
					"entity.A": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg/entity",
						"Name": "A"
					}
				}
			}`,
		},
		{
			name:   "methods and types in a func",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Func_Method"},
			want: `{
				"Exported": true,
				"IsMethod": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Case_Func_Method",
				"File": "util.go",
				"Line": 67,
				"Content": "func Case_Func_Method() {\n\ts := \u0026CaseStruct{}\n\t_ = s.CaseMethod(nil, nil)\n\ts2 := \u0026entity.MyStruct{}\n\t_ = s2.String()\n}",
				"MethodCalls": {
					"s.CaseMethod": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg",
						"Name": "CaseStruct.CaseMethod"
					},
					"s2.String": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg/entity",
						"Name": "MyStruct.String"
					}
				},
				"Types": {
					"CaseStruct": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg",
						"Name": "CaseStruct"
					},
					"entity.MyStruct": {
						"ModPath": "a.b/c",
						"PkgPath": "a.b/c/pkg/entity",
						"Name": "MyStruct"
					}
				}
			}`,
		},
		{
			name:   "global const",
			fields: fields{"a.b/c", "a.b/c/pkg", "Enum3"},
			want: `{
				"IsExported": true,
				"IsConst": true,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Enum3",
				"File": "util.go",
				"Line": 99,
				"Content": "const Enum3 = 3"
			}`,
		},
		{
			name:   "global var1",
			fields: fields{"a.b/c", "a.b/c/pkg", "Var3"},
			want: `{
				"IsExported": true,
				"IsConst": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Var3",
				"File": "util.go",
				"Line": 105,
				"Type": {
					"ModPath": "",
					"PkgPath": "",
					"Name": "int"
				},
				"Content": "var Var3 int = 3"
			}`,
		},
		{
			name:   "global var2",
			fields: fields{"a.b/c", "a.b/c/pkg", "Var7"},
			want: `{
				"IsExported": true,
				"IsConst": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Var7",
				"File": "util.go",
				"Line": 109,
				"Type": {
					"ModPath": "",
					"PkgPath": "",
					"Name": "bool"
				},
				"Content": "var Var7 bool = flag.Bool(\"flag\", false, \"usage\")"
			}`,
		},
		{
			name:   "global var8",
			fields: fields{"a.b/c", "a.b/c/pkg", "Var8"},
			want: `{
				"IsExported": true,
				"IsConst": false,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "Var8",
				"File": "util.go",
				"Line": 110,
				"Type": {
					"ModPath": "github.com/bytedance/sonic@v1.10.2",
					"PkgPath": "github.com/bytedance/sonic",
					"Name": "Config"
				},
				"Content": "var Var8 Config = sonic.Config{}"
			}`,
		},
		{
			name:   "const untyped ",
			fields: fields{"a.b/c", "a.b/c/pkg", "JSON"},
			want: `{
				"IsExported": true,
				"IsConst": true,
				"ModPath": "a.b/c",
				"PkgPath": "a.b/c/pkg",
				"Name": "JSON",
				"File": "util.go",
				"Line": 145,
				"Type": {
					"ModPath": "a.b/c",
					"PkgPath": "a.b/c/pkg",
					"Name": "Type"
				},
				"Content": "const JSON Type = 5"
			}`,
		},
		{
			name:    "refer thirdparty type",
			fields:  fields{"a.b/c", "a.b/c/pkg", "Case_Refer_Type"},
			refered: &Identity{"github.com/bytedance/sonic@v1.10.2", "github.com/bytedance/sonic", "Config"},
			want:    `{"Exported":true,"TypeKind":0,"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic","Name":"Config","File":"api.go","Line":26,"Content":"Config struct {\n    // EscapeHTML indicates encoder to escape all HTML characters \n    // after serializing into JSON (see https://pkg.go.dev/encoding/json#HTMLEscape).\n    // WARNING: This hurts performance A LOT, USE WITH CARE.\n    EscapeHTML                    bool\n\n    // SortMapKeys indicates encoder that the keys of a map needs to be sorted \n    // before serializing into JSON.\n    // WARNING: This hurts performance A LOT, USE WITH CARE.\n    SortMapKeys                   bool\n\n    // CompactMarshaler indicates encoder that the output JSON from json.Marshaler \n    // is always compact and needs no validation \n    CompactMarshaler              bool\n\n    // NoQuoteTextMarshaler indicates encoder that the output text from encoding.TextMarshaler \n    // is always escaped string and needs no quoting\n    NoQuoteTextMarshaler          bool\n\n    // NoNullSliceOrMap indicates encoder that all empty Array or Object are encoded as '[]' or '{}',\n    // instead of 'null'\n    NoNullSliceOrMap              bool\n\n    // UseInt64 indicates decoder to unmarshal an integer into an interface{} as an\n    // int64 instead of as a float64.\n    UseInt64                      bool\n\n    // UseNumber indicates decoder to unmarshal a number into an interface{} as a\n    // json.Number instead of as a float64.\n    UseNumber                     bool\n\n    // UseUnicodeErrors indicates decoder to return an error when encounter invalid\n    // UTF-8 escape sequences.\n    UseUnicodeErrors              bool\n\n    // DisallowUnknownFields indicates decoder to return an error when the destination\n    // is a struct and the input contains object keys which do not match any\n    // non-ignored, exported fields in the destination.\n    DisallowUnknownFields         bool\n\n    // CopyString indicates decoder to decode string values by copying instead of referring.\n    CopyString                    bool\n\n    // ValidateString indicates decoder and encoder to valid string values: decoder will return errors \n    // when unescaped control chars(\\u0000-\\u001f) in the string value of JSON.\n    ValidateString                bool\n\n    // NoValidateJSONMarshaler indicates that the encoder should not validate the output string\n    // after encoding the JSONMarshaler to JSON.\n    NoValidateJSONMarshaler       bool\n}"}`,
		},
		// {
		// 	name:    "refer thirdparty func param",
		// 	fields:  fields{"a.b/c", "a.b/c/pkg", "Case_Refer_Func"},
		// 	refered: &Identity{"github.com/bytedance/sonic@v1.10.2", "github.com/bytedance/sonic/ast", "Node"},
		// 	want:    `{"Exported":true,"TypeKind":0,"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic/ast","Name":"Node","File":"api.go","Line":183,"Content":"type Node struct{}"}`,
		// },
		{
			name:    "refer thirdparty var",
			fields:  fields{"a.b/c", "a.b/c/pkg", "Case_Refer_Var"},
			refered: &Identity{"github.com/bytedance/sonic@v1.10.2", "github.com/bytedance/sonic", "ConfigStd"},
			want:    `{"IsExported":true,"IsConst":false,"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic","Name":"ConfigStd","File":"api.go","Line":78,"Content":"ConfigStd = Config{\n        EscapeHTML : true,\n        SortMapKeys: true,\n        CompactMarshaler: true,\n        CopyString : true,\n        ValidateString : true,\n    }.Froze()"}`,
		},
		{
			name:    "refer thirdparty method",
			fields:  fields{"a.b/c", "a.b/c/pkg", "Case_Refer_Method"},
			refered: &Identity{"github.com/bytedance/sonic@v1.10.2", "github.com/bytedance/sonic", "Config.Froze"},
			want:    `{"Exported":true,"IsMethod":true,"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic","Name":"Config.Froze","File":"compat.go","Line":35,"Content":"func (cfg Config) Froze() API {\n    api := \u0026frozenConfig{Config: cfg}\n    return api\n}","Receiver":{"IsPointer":false,"Type":{"ModPath":"github.com/bytedance/sonic@v1.10.2","PkgPath":"github.com/bytedance/sonic","Name":"Config"},"Name":"Config.Froze"}}`,
		},
		{
			name:   "chain selector",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Chain_Selector"},
			want:   `{"Exported":true,"IsMethod":false,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Case_Chain_Selector","File":"refer.go","Line":45,"Content":"func Case_Chain_Selector() {\n\tvar obj Obj\n\tobj.Call().Call()\n}","MethodCalls":{"obj.Call":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Obj.Call"}},"Types":{"Obj":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Obj"}}}`,
		},
		{
			name:   "closure",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Closure"},
			want:   `{"Exported":true,"IsMethod":false,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Case_Closure","File":"refer.go","Line":50,"Content":"func Case_Closure() {\n\tvar obj Obj\n\tobj.CallFunc(func(o Obj) Obj {\n\t\treturn o.Call()\n\t})\n}","MethodCalls":{"o.Call":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Obj.Call"},"obj.CallFunc":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Obj.CallFunc"}},"Types":{"Obj":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Obj"}}}`,
		},
		{
			name:   "annoymous struct",
			fields: fields{"a.b/c", "a.b/c/pkg", "Case_Annoy_Struct"},
			want:   `{"Exported":true,"TypeKind":0,"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"Case_Annoy_Struct","File":"refer.go","Line":57,"Content":"type Case_Annoy_Struct struct {\n\tA struct {\n\t\tB int\n\t}\n\tC int\n}","SubStruct":{"A":{"ModPath":"a.b/c","PkgPath":"a.b/c/pkg","Name":"_A"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser("a.b/c", "/root/codes/abcoder/testdata", Options{})
			if tt.refered != nil {
				p.opts.ReferCodeDepth = 1
			}
			n, err := p.getNode(NewIdentity(tt.fields.mod, tt.fields.pkg, tt.fields.name))
			if err != nil {
				t.Fatal(err)
			}
			if n == nil {
				t.Fatal("nil get node")
			}
			// spew.Dump(p)
			var j []byte
			if tt.refered == nil {
				j, _ = json.Marshal(n)
			} else {
				ref, err := p.getNode(*tt.refered)
				if err != nil {
					t.Fatal(err)
				}
				j, _ = json.Marshal(ref)
			}
			w := bytes.NewBuffer(nil)
			_ = json.Compact(w, []byte(tt.want))
			require.Equal(t, w.String(), string(j))
		})
	}
}
