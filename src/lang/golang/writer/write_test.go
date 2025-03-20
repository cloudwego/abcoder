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
	"testing"

	"github.com/cloudwego/abcoder/src/uniast"
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
					OutDir:    "../../../../tmp/go_writer/localsession",
					GoVersion: "1.18",
				},
			},
			args:    args{repo: repo},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(tt.fields.Options)
			if err := w.WriteRepo(tt.args.repo); (err != nil) != tt.wantErr {
				t.Errorf("Writer.WriteRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
