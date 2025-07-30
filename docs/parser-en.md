# ABCoder - Language Parser Introduction

ABCoder currently implements Parser based on the [LSP](https://microsoft.github.io/language-server-protocol/) protocol to achieve precise dependency collection and facilitate future multi-language extensions.

## Code Structure

Located under the [lang](/lang) package, including:

- uniast: Golang definitions for unified AST structure
- lsp: LSP protocol processing client, providing interfaces for file parsing, reference lookup, syntax tree parsing, definition lookup, etc., as well as the **generic language specification LanguageSpec interface**
- collect: Responsible for LSP symbol collection and UniAST export, which is the core computation logic
- {language}: Mainly implements the corresponding {language} specification for the lsp#Spec interface. Also includes some specific calling logic for LSP servers

## Operation Process

![lang-parser](../images/lang-parser.png)

1. Identify the language through command line parameters to start the corresponding LSP server and pass initialization parameters
2. Traverse repository files, call the `textDocument/documentSymbol` method to get all symbols for each file. For each symbol:
   1. Call the `textDocument/semanticTokens/range` method to get tokens in the symbol code
   2. Identify valid entity tokens, call `textDocument/definition` to jump to the corresponding symbol location, thus establishing node dependency relationships
3. Repeat step 2 until file processing is complete. Finally convert the collected LSP symbols to UniAST format and output

## Extending Other Language Implementations

Since UniAST is not completely equivalent to LSP, some language-specific behavior interfaces need to be implemented for conversion. Refer to the lang/rust package, generally the following capabilities need to be implemented:

- GetDefaultLSP(): Map user input language to specific lsp.Language and corresponding LSP name
- CheckRepo(): Check user repository status, handle toolchain issues according to language specifications, and return the first file to open by default (for triggering LSP server) and the waiting time for server initialization (determined by repository size)
- **LanguageSpec interface**: Core module for handling non-LSP generic syntax information, such as determining if a token is a standard library symbol, function signature parsing, etc.

### LanguageSpec

```go
// Detailed implementation used for collect LSP symbols and transform them to UniAST
type LanguageSpec interface {
    // initialize a root workspace, and return all modules [modulename=>abs-path] inside
    WorkSpace(root string) (map[string]string, error)

    // give an absolute file path and returns its module name and package path
    // external path should alse be supported
    // FIXEM: some language (like rust) may have sub-mods inside a file, but we still consider it as a unity mod here
    NameSpace(path string) (string, string, error)

    // tells if a file belang to language AST
    ShouldSkip(path string) bool

    // FileImports parse file codes to get its imports
    FileImports(content []byte) ([]uniast.Import, error)

    // return the first declaration token of a symbol, as Type-Name
    DeclareTokenOfSymbol(sym DocumentSymbol) int

    // tells if a token is an AST entity
    IsEntityToken(tok Token) bool

    // tells if a token is a std token
    IsStdToken(tok Token) bool

    // return the SymbolKind of a token
    TokenKind(tok Token) SymbolKind

    // tells if a symbol is a main function
    IsMainFunction(sym DocumentSymbol) bool

    // tells if a symbol is a language symbol (func, type, variable, etc) in workspace
    IsEntitySymbol(sym DocumentSymbol) bool

    // tells if a symbol is public in workspace
    IsPublicSymbol(sym DocumentSymbol) bool

    // declare if the language has impl symbol
    // if it return true, the ImplSymbol() will be called
    HasImplSymbol() bool
    // if a symbol is an impl symbol, return the token index of interface type, receiver type and first-method start (-1 means not found)
    // ortherwise the collector will use FunctionSymbol() as receiver type token index (-1 means not found)
    ImplSymbol(sym DocumentSymbol) (int, int, int)

    // if a symbol is a Function or Method symbol,  return the token index of Receiver (-1 means not found),TypeParameters, InputParameters and Outputs
    FunctionSymbol(sym DocumentSymbol) (int, []int, []int, []int)
}
```

- Rust-parser implementation location: [RustSpec](/lang/rust/spec.go)
