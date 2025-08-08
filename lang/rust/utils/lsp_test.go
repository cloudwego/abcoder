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

package utils

import (
	"strings"
	"testing"
)

func Test_hasIdent(t *testing.T) {
	type args struct {
		text  string
		token string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"simple", args{"ExampleService", "ExampleService"}, true},
		{"simple nonword", args{"ExampleService", "ExampleServicex"}, false},
		{"realworld", args{strings.ToLower(`impl<
                    S: ::volo::service::Service<
                            ::volo_thrift::context::ClientContext,
                            ExampleServiceRequestSend,
                            Response = ::std::option::Option<ExampleServiceResponseRecv>,
                            Error = ::volo_thrift::ClientError,
                        > + Send
                        + Sync
                        + 'static,
                > ExampleServiceGenericClient<S>`), strings.ToLower("ExampleServiceGenericClient")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasIdent(tt.args.text, tt.args.token); got != tt.want {
				t.Errorf("hasIdent() = %v, want %v", got, tt.want)
			}
		})
	}
}
