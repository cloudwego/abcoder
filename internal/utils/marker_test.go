/**
 * Copyright 2024 ByteDance Inc.
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

package utils

import (
	"os"
	"reflect"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestExtractMDCodes(t *testing.T) {
	bs, err := os.ReadFile("../tmp/llm.out")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		resp string
	}
	tests := []struct {
		name          string
		args          args
		wantJs        string
		wantRealLang  uniast.Language
		wantGo        string
		wantRealLang2 uniast.Language
	}{
		{
			name: "test1",
			args: args{
				resp: string(bs),
			},
			wantRealLang:  uniast.Golang,
			wantGo:        "",
			wantJs:        "",
			wantRealLang2: "json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGo, gotRealLang, gotLast := ExtractMDCodes(tt.args.resp, "xx")
			if gotGo != tt.wantJs {
				t.Errorf("ExtractMDCodes() gotGo = %v, want %v", gotGo, tt.wantJs)
			}
			if !reflect.DeepEqual(gotRealLang, tt.wantRealLang) {
				t.Errorf("ExtractMDCodes() gotRealLang = %v, want %v", gotRealLang, tt.wantRealLang)
			}
			gotJs, gotRealLang2, _ := ExtractMDCodes(tt.args.resp[gotLast+1:], "json")
			if gotJs != tt.wantGo {
				t.Errorf("ExtractMDCodes() gotJs = %v, want %v", gotJs, tt.wantGo)
			}
			if !reflect.DeepEqual(gotRealLang2, tt.wantRealLang2) {
				t.Errorf("ExtractMDCodes() gotRealLang2 = %v, want %v", gotRealLang2, tt.wantRealLang2)
			}
		})
	}
}
