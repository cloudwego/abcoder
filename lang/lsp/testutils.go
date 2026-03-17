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
	"log"
	"time"

	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

var clients = make(map[uniast.Language]*LSPClient)

func InitLSPForFirstTest(lang uniast.Language, server string) (*LSPClient, string, error) {
	testdata := testutils.FirstTest(string(lang))
	if client, exists := clients[lang]; exists {
		return client, testdata, nil
	}

	client, err := NewLSPClient(testdata, "", 0, ClientOptions{
		Server:   server,
		Language: uniast.Language(lang),
		Verbose:  true,
	})
	if err != nil {
		log.Fatalf("Failed to initialize %s LSP client: %v", lang, err)
		return nil, "", err
	}
	clients[lang] = client
	time.Sleep(5 * time.Second) // wait for LSP server to be ready
	return client, testdata, nil
}
