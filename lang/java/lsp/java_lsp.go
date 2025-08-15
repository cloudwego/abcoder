package lsp

import (
	"context"

	"github.com/cloudwego/abcoder/lang/lsp"
	golsp "github.com/sourcegraph/go-lsp"
)

// JavaProvider implements the LanguageServiceProvider for Java.
type JavaProvider struct{}

// jdtHover is a custom struct to handle the hover result from JDT LS
// It supports both MarkupContent object and simple string formats
type jdtHover struct {
	Contents interface{} `json:"contents"`
	Range    *lsp.Range  `json:"range,omitempty"`
}

func (p *JavaProvider) Hover(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, line, character int) (*golsp.Hover, error) {
	var result jdtHover // Use the custom struct to unmarshal
	err := cli.Call(ctx, "textDocument/hover", golsp.TextDocumentPositionParams{
		TextDocument: golsp.TextDocumentIdentifier{URI: golsp.DocumentURI(uri)},
		Position:     golsp.Position{Line: line, Character: character},
	}, &result)
	if err != nil {
		return nil, err
	}

	// Handle different response formats
	var content string

	// Try to parse as MarkupContent object
	if contentsMap, isMap := result.Contents.(map[string]interface{}); isMap {
		if value, exists := contentsMap["value"]; exists {
			if strValue, isString := value.(string); isString {
				content = strValue
			}
		}
	} else if strContent, isString := result.Contents.(string); isString {
		// Handle simple string response
		content = strContent
	}

	// Convert the JDT-specific hover result to the standard lsp.Hover type.
	standardHover := &golsp.Hover{
		Contents: []golsp.MarkedString{
			{
				Language: "java",
				Value:    content,
			},
		},
		Range: &golsp.Range{},
	}

	return standardHover, nil
}

func (p *JavaProvider) Implementation(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.Location, error) {
	var result []lsp.Location
	err := cli.Call(ctx, "textDocument/implementation", golsp.TextDocumentPositionParams{
		TextDocument: golsp.TextDocumentIdentifier{URI: golsp.DocumentURI(uri)},
		Position:     golsp.Position(pos),
	}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *JavaProvider) WorkspaceSearchSymbols(ctx context.Context, cli *lsp.LSPClient, query string) ([]golsp.SymbolInformation, error) {
	req := golsp.WorkspaceSymbolParams{
		Query: query,
	}
	var resp []golsp.SymbolInformation
	if err := cli.Call(ctx, "workspace/symbol", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PrepareTypeHierarchy performs a textDocument/prepareTypeHierarchy request.
func (p *JavaProvider) PrepareTypeHierarchy(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.TypeHierarchyItem, error) {
	params := golsp.TextDocumentPositionParams{
		TextDocument: golsp.TextDocumentIdentifier{URI: golsp.DocumentURI(uri)},
		Position:     golsp.Position(pos),
	}

	var result []lsp.TypeHierarchyItem
	err := cli.Call(ctx, "textDocument/prepareTypeHierarchy", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TypeHierarchySupertypes requests the supertypes of a symbol.
func (p *JavaProvider) TypeHierarchySupertypes(ctx context.Context, cli *lsp.LSPClient, item lsp.TypeHierarchyItem) ([]lsp.TypeHierarchyItem, error) {
	params := struct {
		Item lsp.TypeHierarchyItem `json:"item"`
	}{
		Item: item,
	}
	var result []lsp.TypeHierarchyItem
	err := cli.Call(ctx, "typeHierarchy/supertypes", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TypeHierarchySubtypes requests the subtypes of a symbol.
func (p *JavaProvider) TypeHierarchySubtypes(ctx context.Context, cli *lsp.LSPClient, item lsp.TypeHierarchyItem) ([]lsp.TypeHierarchyItem, error) {
	params := struct {
		Item lsp.TypeHierarchyItem `json:"item"`
	}{
		Item: item,
	}
	var result []lsp.TypeHierarchyItem
	err := cli.Call(ctx, "typeHierarchy/subtypes", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
