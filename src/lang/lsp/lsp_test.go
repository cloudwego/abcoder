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

package lsp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDocumentSymbol_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		fields  DocumentSymbol
		want    []byte
		wantErr bool
	}{
		{
			name:    "rust2",
			fields:  DocumentSymbol{},
			want:    []byte(`{"name":"","kind":0,"tags":null,"location":":1:1-1:1","children":null,"text":"","tokens":null}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(&tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("DocumentSymbol.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DocumentSymbol.MarshalJSON() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
