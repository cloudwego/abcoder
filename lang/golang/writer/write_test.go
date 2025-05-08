/**
 * Copyright 2025 ByteDance Inc.
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

package writer

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestWriter_WriteRepo(t *testing.T) {
	repo, err := uniast.LoadRepo("../../../../tmp_compress/localsession.json")
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		Options Options
	}
	type args struct {
		repo *uniast.Repository
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
				Options: Options{
					CompilerPath: "1.18",
				},
			},
			args:    args{repo: repo},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(tt.fields.Options)
			if err := w.WriteRepo(tt.args.repo, "../../../../tmp/localsession2"); (err != nil) != tt.wantErr {
				t.Errorf("Writer.WriteRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatcher_PatchImports(t *testing.T) {
	data, err := os.ReadFile("../../../../tmp/localsession/gls.go")
	if err != nil {
		t.Errorf("fail read file %v", err)
		return
	}
	alias1 := string("_")
	data1 := bytes.Replace(data, []byte(`import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
`), []byte(`import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "runtime"
)
`), 1)
	data2, err := os.ReadFile("../../../../tmp/localsession/backup/xx_test.go")
	if err != nil {
		t.Errorf("fail read file %v", err)
		return
	}
	data2 = bytes.Replace(data2, []byte(`package backup
`), []byte(`package backup
import "fmt"
`), 1)

	type args struct {
		file *uniast.File
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// {
		// 	name: "empty new",
		// 	args: args{
		// 		file: &uniast.File{
		// 			Name:    "gls.go",
		// 			Imports: []uniast.Import{},
		// 			Path:    "gls.go",
		// 		},
		// 	},
		// 	want:    data,
		// 	wantErr: false,
		// },
		// {
		// 	name: "empty old",
		// 	args: args{
		// 		file: &uniast.File{
		// 			Name: "backup/xx_test.go",
		// 			Imports: []uniast.Import{
		// 				{
		// 					Path:  `"fmt"`,
		// 					Alias: nil,
		// 				},
		// 			},
		// 			Path: "backup/xx_test.go",
		// 		},
		// 	},
		// 	want:    data2,
		// 	wantErr: false,
		// },
		{
			name: "add",
			args: args{
				file: &uniast.File{
					Imports: []uniast.Import{
						{
							Path:  `runtime`,
							Alias: &alias1,
						},
					},
					Path: "gls.go",
				},
			},
			want:    data1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		p := NewWriter(Options{})
		t.Run(tt.name, func(t *testing.T) {
			old, err := os.ReadFile(tt.args.file.Path)
			if err != nil {
				t.Errorf("fail read file %v", err)
				return
			}
			got, err := p.PatchImports(tt.args.file.Imports, old)
			if (err != nil) != tt.wantErr {
				t.Errorf("Patcher.PatchImports() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Patcher.PatchImports() = %s, want %s", got, tt.want)
			}
		})
	}
}
