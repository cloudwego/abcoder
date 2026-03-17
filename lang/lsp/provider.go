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
	"context"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// LanguageServiceProvider defines methods for language-specific LSP features.
// It allows for extending the base LSPClient with language-specific capabilities
// without creating circular dependencies.
type LanguageServiceProvider interface {
	// Hover provides hover information for a given position.
	// Implementations may have custom logic to parse results from different language servers.
	Hover(ctx context.Context, cli *LSPClient, uri DocumentURI, line, character int) (*Hover, error)

	// Implementation finds implementations of a symbol.
	Implementation(ctx context.Context, cli *LSPClient, uri DocumentURI, pos Position) ([]Location, error)

	// WorkspaceSymbols searches for symbols in the workspace.
	WorkspaceSearchSymbols(ctx context.Context, cli *LSPClient, query string) ([]SymbolInformation, error)

	// PrepareTypeHierarchy prepares a type hierarchy for a given position.
	PrepareTypeHierarchy(ctx context.Context, cli *LSPClient, uri DocumentURI, pos Position) ([]TypeHierarchyItem, error)

	// TypeHierarchySupertypes gets the supertypes of a type hierarchy item.
	TypeHierarchySupertypes(ctx context.Context, cli *LSPClient, item TypeHierarchyItem) ([]TypeHierarchyItem, error)

	// TypeHierarchySubtypes gets the subtypes of a type hierarchy item.
	TypeHierarchySubtypes(ctx context.Context, cli *LSPClient, item TypeHierarchyItem) ([]TypeHierarchyItem, error)
}

var providers = make(map[uniast.Language]LanguageServiceProvider)

// RegisterProvider makes a LanguageServiceProvider available for a given language.
// This function should be called from the init() function of a language-specific package.
func RegisterProvider(lang uniast.Language, provider LanguageServiceProvider) {
	if _, dup := providers[lang]; dup {
		// Or maybe log a warning
		return
	}
	providers[lang] = provider
}

// GetProvider returns the registered LanguageServiceProvider for a given language.
// It returns nil if no provider is registered for the language.
func GetProvider(lang uniast.Language) LanguageServiceProvider {
	return providers[lang]
}
