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
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	. "github.com/cloudwego/abcoder/lang/uniast"
)

func Test_goParser_ParseRepo(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test",
			fields: fields{
				modName:     "github.com/cloudwego/localsession",
				homePageDir: "../../../../tmp/localsession",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs, _ := filepath.Abs(tt.fields.homePageDir)
			println(abs)
			p := newGoParser(tt.fields.modName, tt.fields.homePageDir, Options{
				ReferCodeDepth: 1,
				NeedTest:       true,
			})
			r, err := p.ParseRepo()
			if err != nil {
				t.Fatal(err)
			}
			r.BuildGraph()
			// spew.Dump(p)
			pj, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			_ = pj
			_ = os.WriteFile("ast.json", pj, 0644)
			n, err := p.getNode(NewIdentity("github.com/cloudwego/localsession", "github.com/cloudwego/localsession/backup", "RecoverCtxOnDemands"))
			if err != nil {
				t.Fatal(err)
			}
			jf, err := json.MarshalIndent(n, "", "  ")
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			os.WriteFile("node.json", jf, 0644)
		})
	}
}

func Test_goParser_ParseDirs(t *testing.T) {
	type args struct {
		modName     string
		homePageDir string
		pkg         string
		opts        Options
	}
	tests := []struct {
		name    string
		args    args
		wantRet map[string]*Function
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				homePageDir: "../../../../testdata/golang",
				modName:     "a.b/c",
				pkg:         "a.b/c/cmd",
				opts: Options{
					ReferCodeDepth: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newGoParser(tt.args.modName, tt.args.homePageDir, tt.args.opts)
			_, err := p.ParseRepo()
			if (err != nil) != tt.wantErr {
				t.Errorf("goParser.ParseDirs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			out, err := json.MarshalIndent(p.repo, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			_ = out
			println(string(out))
		})
	}
}

func TestGoAst(t *testing.T) {
	src := `
package parse

type Struct struct {
	A struct{
		B int
	}
}
	`
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(node, func(n ast.Node) bool {
		fmt.Printf("%#v\n", n)
		if sel, ok := n.(*ast.SelectorExpr); ok {
			println("selector:", string(GetRawContent(fset, []byte(src), sel, false)))
		}
		if stru, ok := n.(*ast.StructType); ok {
			println("struct:", string(GetRawContent(fset, []byte(src), stru, true)))
		}
		return true
	})
}

func Test_goParser_ParseNode(t *testing.T) {
	type fields struct {
		modName     string
		homePageDir string
	}
	type args struct {
		pkgPath string
		name    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				modName:     "github.com/cloudwego/localsession",
				homePageDir: "../../../../../tmp/localsession",
			},
			args: args{
				pkgPath: "github.com/modern-go/gls",
				name:    "DeleteGls",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.fields.modName, tt.fields.homePageDir, Options{})
			got, err := p.ParseNode(tt.args.pkgPath, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("goParser.ParseNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			out, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			println(string(out))
		})
	}
}
