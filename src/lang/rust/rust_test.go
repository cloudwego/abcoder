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

package rust

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	lsp "github.com/cloudwego/abcoder/src/lang/lsp"
)

func TestRustSpec_NameSpace(t *testing.T) {
	type args struct {
		root string
	}
	type nameSpace struct {
		path    string
		wantMod string
		wantPkg string
	}
	tests := []struct {
		name      string
		args      args
		want      map[string]string
		nameSpace []nameSpace
		wantErr   bool
	}{
		{name: "",
			args: args{"/root/codes/abcoder"},
			nameSpace: []nameSpace{
				{"/root/codes/abcoder/src/lib.rs", "ABCoder", "ABCoder"},
				{"/root/codes/abcoder/src/repo.rs", "ABCoder", "ABCoder::repo"},
				{"/root/codes/abcoder/src/config/mod.rs", "ABCoder", "ABCoder::config"},
				{"/root/codes/abcoder/src/utils/cmd.rs", "ABCoder", "ABCoder::utils::cmd"},
				{"/root/codes/abcoder/testdata/rust2/src/main.rs", "rust2", "rust2"},
				{"/root/codes/abcoder/testdata/rust2/src/entity/mod.rs", "rust2", "rust2::entity"},
				{"/root/.cargo/registry/src/xxx/byted-env-0.2.8/src/lib.rs", "byted-env@0.2.8", "byted-env"},
				{"/root/.cargo/registry/src/xxx/byted-env-0.2.8/src/idc/mod.rs", "byted-env@0.2.8", "byted-env::idc"},
				{"/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/lib/rustlib/src/rust/library/alloc/src/alloc.rs", "", "alloc::alloc"},
				{"/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/lib/rustlib/src/rust/library/std/src/f32.rs", "std", "std::f32"},
			},
			want:    map[string]string{"ABCoder": "/root/codes/abcoder/src", "rust2": "/root/codes/abcoder/testdata/rust2/src"},
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRustSpec()
			got, err := c.WorkSpace(tt.args.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("RustSpec.CollectModules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RustSpec.CollectModules() = %v, want %v", got, tt.want)
			}
			// test namespace
			for _, ns := range tt.nameSpace {
				fmt.Printf("test: %#v\n", ns)
				gotMod, gotPkg, err := c.NameSpace(ns.path)
				if err != nil {
					t.Errorf("RustSpec.NameSpace() error = %v", err)
					return
				}
				if gotMod != ns.wantMod {
					t.Errorf("RustSpec.NameSpace() crate get %v, want %v", gotMod, ns.wantMod)
				}
				if gotPkg != ns.wantPkg {
					t.Errorf("RustSpec.NameSpace() mod get %v, want %v", gotPkg, ns.wantPkg)
				}
			}
		})
	}
}

func getData(path string, key string) *lsp.DocumentSymbol {
	f, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(f, &obj); err != nil {
		return nil
	}

	var js string
	for k, v := range obj {
		if strings.HasPrefix(k, key) {
			js = string(v)
			break
		}
	}
	var ret lsp.DocumentSymbol
	if err := json.Unmarshal([]byte(js), &ret); err != nil {
		return nil
	}
	return &ret
}

func TestRustSpec_FunctionSymbol(t *testing.T) {
	var file = "../testdata/symbols_rust2-save.json"
	tests := []struct {
		name  string
		args  lsp.DocumentSymbol
		want  int
		want1 int
		want2 int
	}{
		{
			name:  "rust2-write_to_output",
			args:  *getData(file, "write_to_output Function"),
			want:  2,
			want1: 1,
			want2: 3,
		},
		{
			name:  "rust2-add",
			args:  *getData(file, "add Function file:///root/codes/abcoder/testdata/rust2/src/entity/mod.rs:16:1-18:2"),
			want:  0,
			want1: 2,
			want2: 1,
		},
		{
			name:  "rust2-my_trait",
			args:  *getData(file, "my_trait Method file:///root/codes/abcoder/testdata/rust2/src/entity/mod.rs:36:5-36:33"),
			want:  0,
			want1: 0,
			want2: 0,
		},
		{
			name:  "main Function",
			args:  *getData(file, "main Function"),
			want:  0,
			want1: 0,
			want2: 0,
		},
		{
			name:  "apply Function",
			args:  *getData(file, "apply_closure Function"),
			want:  3,
			want1: 2,
			want2: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRustSpec()
			_, err := c.WorkSpace("/root/codes/abcoder")
			if err != nil {
				t.Errorf("RustSpec.WorkSpace() error = %v", err)
			}
			rec, got, got1, got2 := c.FunctionSymbol(tt.args)
			if rec != -1 {
				t.Logf("FunctionSymbol: %#v", rec)
			}
			if len(got) != tt.want {
				t.Errorf("RustSpec.FunctionSymbol() got = %v, want %v", got, tt.want)
			}
			if len(got1) != tt.want1 {
				t.Errorf("RustSpec.FunctionSymbol() got1 = %v, want %v", got1, tt.want1)
			}
			if len(got2) != tt.want2 {
				t.Errorf("RustSpec.FunctionSymbol() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func BenchmarkRustSpec_FunctionSymbol(b *testing.B) {
	var file = "../../testdata/symbols_rust2-save.json"
	c := NewRustSpec()
	_, err := c.WorkSpace("/root/codes/abcoder")
	if err != nil {
		b.Errorf("RustSpec.WorkSpace() error = %v", err)
	}
	ds := *getData(file, "write_to_output Function")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.FunctionSymbol(ds)
	}
}
