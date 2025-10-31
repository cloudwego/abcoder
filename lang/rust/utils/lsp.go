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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/abcoder/lang/log"
	lsp "github.com/cloudwego/abcoder/lang/lsp"
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
func GetRawSymbol(root, file, mod, name string, receiver string, caseInsensitive bool) *lsp.DocumentSymbol {
	cli := GetLSPClient(root)
	sym, err := getSymbol(cli, root, file, mod, name, receiver, caseInsensitive)
	if err != nil {
		log.Error("get symbol for %s failed, err: %v", name, err)
		return nil
	}

	return sym
}

// 查找符号对应的代码
// root: 项目根目录
// file: 文件路径
// mod: 命名空间，空表示本文件根
// name: 符号名
// receiver: method接收者，为空表示不是method
func GetSymbol(root, file, mod, name string, receiver string, caseInsensitive bool) string {
	cli := GetLSPClient(root)
	sym, err := getSymbol(cli, root, file, mod, name, receiver, caseInsensitive)
	if err != nil {
		log.Error("get symbol for %s failed, err: %v", name, err)
		return ""
	}
	if sym == nil {
		return ""
	}
	text, _ := cli.Locate(sym.Location)
	return text
}

func getSymbol(cli *lsp.LSPClient, root, file, mod, name string, receiver string, caseInsensitive bool) (*lsp.DocumentSymbol, error) {
	syms, err := cli.FileStructure(context.Background(), lsp.NewURI(file))
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("can not find symbol for %s", name)
	}

	return sym, nil
}

// 查找符号对应的源码以及文件行号
// root: 项目根目录
// file: 文件路径
// mod: 命名空间，空表示本文件根
// name: 符号名
// receiver: method接收者，为空表示不是method
func GetSymbolContentAndLocation(root, file, mod, name string, receiver string, caseInsensitive bool) (string, [2]int) {
	cli := GetLSPClient(root)
	sym, err := getSymbol(cli, root, file, mod, name, receiver, caseInsensitive)
	if err != nil {
		log.Error("get symbol for %s failed, err: %v", name, err)
		return "", [2]int{}
	}
	if sym == nil {
		return "", [2]int{}
	}
	text, _ := cli.Locate(sym.Location)
	return text, [2]int{sym.Location.Range.Start.Line, sym.Location.Range.End.Line}
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
	// match regex: \<token\> in text
	if token == "" {
		log.Error("token cannot be empty")
		return false
	}
	ptn := regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`)
	return ptn.MatchString(text)
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
