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
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lsp "github.com/cloudwego/abcoder/src/lang/lsp"
)

var DefaultLSPInitTime = time.Second * 60

var clis sync.Map

func GetLSPClient(root string) *lsp.LSPClient {
	if cli, ok := clis.Load(root); ok {
		return cli.(*lsp.LSPClient)
	}
	cli, err := lsp.NewLSPClient(root, filepath.Join(root, "Cargo.toml"), DefaultLSPInitTime, lsp.ClientOptions{
		Server:   "rust-analyzer",
		Language: "rust",
		Verbose:  true,
	})
	if err != nil {
		panic(err)
	}
	clis.Store(root, cli)
	return cli
}

// 查找符号对应的代码
// root: 项目根目录
// file: 文件路径
// mod: 命名空间，空表示本文件根
// name: 符号名
// receiver: method接收者，为空表示不是method
func GetSymbol(root, file, mod, name string, receiver string, caseInsensitive bool) string {
	cli := GetLSPClient(root)
	syms, err := cli.FileStructure(context.Background(), lsp.NewURI(file))
	if err != nil {
		return ""
	}

	var sym *lsp.DocumentSymbol
	if mod != "" {
		// recursive search mod to the deepest
		syms = dfsSyms(syms, strings.Split(mod, "::"))
	}
	if receiver != "" {
		for _, s := range syms {
			if s.Kind == lsp.SKObject && ((caseInsensitive && (hasIdent(strings.ToLower(s.Name), strings.ToLower(receiver)))) || hasIdent(s.Name, receiver)) {
				if name == "" {
					sym = s
					goto finally
				}
				syms = s.Children
				for _, s := range syms {
					if s.Name == name || (caseInsensitive && strings.EqualFold(s.Name, name)) {
						sym = s
						goto finally
					}
				}
			}
		}
	} else {
		for _, s := range syms {
			if s.Name == name || (caseInsensitive && strings.EqualFold(s.Name, name)) {
				sym = s
				break
			}
		}
	}

finally:
	if sym == nil {
		return ""
	}
	text, _ := cli.Locate(sym.Location)
	return text
}

// NOTICE: 为了提供容错率，这里只是简单查找是否包含token，不做严格的标识符检查
func hasIdent(text string, token string) bool {
	// l := len(text)
	// idx := strings.Index(text, token)
	// if idx == -1 {
	// 	return false
	// }
	// for idx < l && idx >= 0 {
	// 	left := idx == 0 || (len(text) > idx-1 && !isAlpha(text[idx-1]) && text[idx-1] != '_')
	// 	right := idx+len(token) == l || (len(text) > idx+len(token) && !isAlpha(text[idx+len(token)]) && text[idx+len(token)] != '_')
	// 	if left && right {
	// 		return true
	// 	} else {
	// 		text = text[idx+len(token):]
	// 		idx = strings.Index(text, token)
	// 		continue
	// 	}
	// }
	return strings.Contains(text, token)
}

// func isAlpha(r byte) bool {
// 	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
// }

func dfsSyms(syms []*lsp.DocumentSymbol, paths []string) []*lsp.DocumentSymbol {
	for _, sym := range syms {
		if sym.Name == paths[0] {
			if len(paths) == 1 {
				return sym.Children
			}
			return dfsSyms(sym.Children, paths[1:])
		}
	}
	return nil
}
