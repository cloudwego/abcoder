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

package tool

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestMCPClient(t *testing.T) {
	cli, err := NewMCPClient(MCPConfig{
		Type:   MCPTypeStdio,
		Comand: "npx",
		Args: []string{
			"-y",
			"@modelcontextprotocol/server-sequential-thinking",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	tools, err := cli.GetTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", tools)
}

func TestGetGitTools(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []Tool
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGitTools(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGitTools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGitTools() = %v, want %v", got, tt.want)
			}
		})
	}
}
