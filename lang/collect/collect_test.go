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
	"github.com/cloudwego/abcoder/lang/java"
	javaLsp "github.com/cloudwego/abcoder/lang/java/lsp"
	"os"
	"testing"

	"path/filepath"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestCollector_CollectByTreeSitter_Java(t *testing.T) {
	log.SetLogLevel(log.DebugLevel)
	javaTestCase := "../../testdata/java/1_advanced"

	t.Run("javaCollect", func(t *testing.T) {

		lsp.RegisterProvider(uniast.Java, &javaLsp.JavaProvider{})

		openfile, wait := java.CheckRepo(javaTestCase)
		l, s := java.GetDefaultLSP(make(map[string]string))
		client, err := lsp.NewLSPClient(javaTestCase, openfile, wait, lsp.ClientOptions{
			Server:   s,
			Language: l,
			Verbose:  false,
		})

		c := NewCollector(javaTestCase, client)
		c.Language = uniast.Java
		_, err = c.ScannerByTreeSitter(context.Background())
		if err != nil {
			t.Fatalf("Collector.CollectByTreeSitter() failed = %v\n", err)
		}

		if len(c.files) == 0 {
			t.Fatalf("Expected have file, but got %d", len(c.files))
		}

		expectedFile := filepath.Join(javaTestCase, "/src/main/java/org/example/test.json")
		if _, ok := c.files[expectedFile]; ok {
			t.Fatalf("Expected file %s not found", expectedFile)
		}
	})
}
func TestCollector_Collect(t *testing.T) {
	log.SetLogLevel(log.DebugLevel)
	rustLSP, rustTestCase, err := lsp.InitLSPForFirstTest(uniast.Rust, "rust-analyzer")
	if err != nil {
		t.Fatalf("Failed to initialize rust LSP client: %v", err)
	}
	defer rustLSP.Close()

	t.Run("rustCollect", func(t *testing.T) {
		c := NewCollector(rustTestCase, rustLSP)
		c.LoadExternalSymbol = true
		err := c.Collect(context.Background())
		if err != nil {
			t.Fatalf("Collector.Collect() failed = %v\n", err)
		}

		outdir := testutils.MakeTmpTestdir(true)
		marshals := []struct {
			val  any
			name string
		}{
			{&c.syms, "symbols"},
			{&c.deps, "deps"},
			{&c.funcs, "funcs"},
			{&c.vars, "vars"},
			{&c.repo, "repo"},
		}
		for _, m := range marshals {
			js, err := json.Marshal(m.val)
			if err != nil {
				t.Fatalf("Marshal %s failed: %v", m.name, err)
			}
			if err := os.WriteFile(outdir+"/"+m.name+".json", js, 0644); err != nil {
				t.Fatalf("Write json %s failed: %v", m.name, err)
			}
		}
	})
}
