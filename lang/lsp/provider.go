package lsp

import (
	"context"
	"github.com/cloudwego/abcoder/lang/uniast"
	golsp "github.com/sourcegraph/go-lsp"
)

// LanguageServiceProvider defines methods for language-specific LSP features.
// It allows for extending the base LSPClient with language-specific capabilities
// without creating circular dependencies.
type LanguageServiceProvider interface {
	// Hover provides hover information for a given position.
	// Implementations may have custom logic to parse results from different language servers.
	Hover(ctx context.Context, cli *LSPClient, uri DocumentURI, line, character int) (*golsp.Hover, error)

	// Implementation finds implementations of a symbol.
	Implementation(ctx context.Context, cli *LSPClient, uri DocumentURI, pos Position) ([]Location, error)

	// WorkspaceSymbols searches for symbols in the workspace.
	WorkspaceSearchSymbols(ctx context.Context, cli *LSPClient, query string) ([]golsp.SymbolInformation, error)

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
