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
	"testing"
	"time"

	"github.com/cloudwego/abcoder/lang/log"
)

func TestCheckRepo(t *testing.T) {
	type args struct {
		repo string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 time.Duration
	}{
		{"rust2", args{"/home/duanyi.aster/Rust/ABCoder/testdata/rust2"}, "/home/duanyi.aster/Rust/ABCoder/testdata/rust2/src/main.rs", time.Second * 15},
		{"live", args{"/home/duanyi.aster/Rust/ABCoder/tmp/live"}, "/home/duanyi.aster/Rust/ABCoder/tmp/live/src/env.rs", time.Minute * 1},
	}

	log.SetLogLevel(log.DebugLevel)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := CheckRepo(tt.args.repo)
			if got != tt.want {
				t.Errorf("CheckRepo() got = %v, want %v", got, tt.want)
			}
			if got1 < tt.want1 {
				t.Errorf("CheckRepo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
