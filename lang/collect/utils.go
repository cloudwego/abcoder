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

import "github.com/cloudwego/abcoder/lang/lsp"

func isFuncLike(sk lsp.SymbolKind) bool {
	return sk == lsp.SKFunction || sk == lsp.SKMethod
	// SKConstructor ?
}

func isTypeLike(sk lsp.SymbolKind) bool {
	return sk == lsp.SKClass || sk == lsp.SKStruct || sk == lsp.SKInterface || sk == lsp.SKEnum
}

func isVarLike(sk lsp.SymbolKind) bool {
	return sk == lsp.SKVariable || sk == lsp.SKConstant
	// sk == lsp.SKField || sk == lsp.SKProperty ?
}
