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
	"reflect"
	"testing"

	"github.com/cloudwego/abcoder/lang/testutils"
)

func TestRustSpec_NameSpaceInternal(t *testing.T) {
	type args struct {
		root string
	}
	type nameSpace struct {
		relPath string
		wantMod string
		wantPkg string
	}
	rustTestRoot := testutils.FirstTest("rust")
	tests := []struct {
		name      string
		args      args
		nameSpace []nameSpace
		want      map[string]string
		wantErr   bool
	}{
		{name: "",
			args: args{rustTestRoot},
			nameSpace: []nameSpace{
				{"/src/main.rs", "rust2", "rust2"},
				{"/src/entity/mod.rs", "rust2", "rust2::entity"},
				{"/src/entity/func.rs", "rust2", "rust2::entity::func"},
				{"/src/entity/inter.rs", "rust2", "rust2::entity::inter"},
				// {"/root/.cargo/registry/src/xxx/byted-env-0.2.8/src/lib.rs", "byted-env@0.2.8", "byted-envs"},
				// 				{"/root/.cargo/registry/src/xxx/byted-env-0.2.8/src/idc/mod.rs", "byted-env@0.2.8", "byted-env::idc"},
				// 				{"/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/lib/rustlib/src/rust/library/alloc/src/alloc.rs", "", "alloc::alloc"},
				// {"/root/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/lib/rustlib/src/rust/library/std/src/f32.rs", "std", "std::f32"},
			},
			want: map[string]string{
				"rust2": rustTestRoot + "/src"},
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRustSpec()
			// Workspace
			got, err := c.WorkSpace(tt.args.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("RustSpec.WorkSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RustSpec.WorkSpace() got = %v, want %v", got, tt.want)
			}
			// Namespace
			for _, ns := range tt.nameSpace {
				gotMod, gotPkg, err := c.NameSpace(tt.args.root + ns.relPath)
				if err != nil {
					t.Errorf("RustSpec.NameSpace() error = %v", err)
					return
				}
				if gotMod != ns.wantMod {
					t.Errorf("RustSpec.NameSpace() crate got %v, want %v", gotMod, ns.wantMod)
				}
				if gotPkg != ns.wantPkg {
					t.Errorf("RustSpec.NameSpace() mod got %v, want %v", gotPkg, ns.wantPkg)
				}
			}
		})
	}
}
