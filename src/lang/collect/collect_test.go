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

package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/lsp"
	parse "github.com/cloudwego/abcoder/src/uniast"
)

var testroot = "../../../testdata"

func TestCollector_Collect(t *testing.T) {
	root := testroot + "/rust2"
	log.SetLogLevel(log.DebugLevel)
	rustLSP, err := lsp.NewLSPClient(root, root+"/src/main.rs", time.Second*15, lsp.ClientOptions{
		Server:   "rust-analyzer",
		Language: "rust",
		Verbose:  true,
	})
	if err != nil {
		fmt.Printf("Failed to initialize rust LSP client: %v", err)
	}
	defer rustLSP.Close()

	tests := []struct {
		name    string
		want    *parse.Repository
		wantErr bool
	}{
		{
			name:    "rust",
			want:    &parse.Repository{},
			wantErr: false,
		},
	}
	dir := testroot
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollector(root, rustLSP)
			c.LoadExternalSymbol = true
			err := c.Collect(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Collector.Collect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			js1, err := json.Marshal(c.syms)
			if err != nil {
				t.Fatalf("Marshal symbols failed: %v", err)
			}
			if err := os.WriteFile(dir+"/symbols.json", js1, 0644); err != nil {
				t.Fatalf("Write json failed: %v", err)
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("Collector.Collect() = %#v, want %#v", got, tt.want)
			// }
			// for sym, content := range c.symbols {
			// 	if sym.Name == "add" {
			// 		t.Logf("symbol: %#v, content:%s", sym, content)
			// 	}
			// }
			js3, err := json.Marshal(c.deps)
			if err != nil {
				t.Fatalf("Marshal deps failed: %v", err)
			}
			if err := os.WriteFile(dir+"/deps.json", js3, 0644); err != nil {
				t.Fatalf("Write json failed: %v", err)
			}
			js4, err := json.Marshal(c.funcs)
			if err != nil {
				t.Fatalf("Marshal methods failed: %v", err)
			}
			if err := os.WriteFile(dir+"/funcs.json", js4, 0644); err != nil {
				t.Fatalf("Write json failed: %v", err)
			}
			js5, err := json.Marshal(c.vars)
			if err != nil {
				t.Fatalf("Marshal methods failed: %v", err)
			}
			if err := os.WriteFile(dir+"/vars.json", js5, 0644); err != nil {
				t.Fatalf("Write json failed: %v", err)
			}

			repo, err := c.Export(context.Background())
			if err != nil {
				t.Fatalf("export repo failed: %v", err)
			}
			js6, err := json.Marshal(repo)
			if err != nil {
				t.Fatalf("Marshal methods failed: %v", err)
			}
			if err := os.WriteFile(dir+"/repo.json", js6, 0644); err != nil {
				t.Fatalf("Write json failed: %v", err)
			}
		})
	}
}
