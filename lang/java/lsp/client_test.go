package lsp

import (
	"context"
	"fmt"
	"github.com/cloudwego/abcoder/lang/uniast"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/abcoder/lang/java"
	"github.com/cloudwego/abcoder/lang/lsp"
	"github.com/stretchr/testify/require"
)

// TestJavaLSPConnection demonstrates connecting to an external Java LSP server.
func TestJavaLSPConnection(t *testing.T) {

	projectRoot := "../../../testdata/java/0_simple"
	ctx := context.Background()

	openfile, wait := java.CheckRepo(projectRoot)
	l, s := java.GetDefaultLSP()
	lsp.RegisterProvider(uniast.Java, &JavaProvider{})

	lspClient, err := lsp.NewLSPClient(projectRoot, openfile, wait, lsp.ClientOptions{
		Server:   s,
		Language: l,
		Verbose:  false,
	})
	if err != nil {
		t.Fatalf("init lspclient failed = %v\n", err)
	}

	// --- Step 1 & 2: Connect and initialize the LSP client ---
	require.NoError(t, err, "Failed to initialize LSP client")

	lspClient.InitFiles()

	// --- Step 3: Open the document to prepare for analysis ---
	fileToAnalyze := "../../../testdata/java/0_simple/HelloWorld.java"
	fileURI := lsp.NewURI(fileToAnalyze)

	_, err = lspClient.DidOpen(ctx, fileURI)
	require.NoError(t, err, "textDocument/didOpen notification failed")

	// --- Step 4: Send the 'textDocument/documentSymbol' request to get the syntax tree ---
	symbols, err := lspClient.DocumentSymbols(ctx, fileURI)
	require.NoError(t, err, "textDocument/documentSymbol request failed")

	// --- Step 5: Process and print the response ---
	require.NotEmpty(t, symbols, "Expected to receive symbols, but got none")

	fmt.Println("Successfully retrieved document symbols for HelloWorld.java:")
	for k, s := range symbols {
		printSymbol(k, s, 0)
	}

	// --- Step 6: Send the 'textDocument/hover' request to get method type info ---
	hoverResult, err := lspClient.Hover(ctx, fileURI, 11, 25)
	require.NoError(t, err, "textDocument/hover request failed")

	fmt.Println("\n--- Hover Result for testFunction ---")
	require.NotEmpty(t, hoverResult.Contents, "Expected hover to have content")
	fmt.Printf("Hover Content: %s\n", hoverResult.Contents[0].Value)
	fmt.Println("-------------------------------------")
}

// printSymbol is a helper to recursively print the symbol structure.
func printSymbol(r lsp.Range, symbol *lsp.DocumentSymbol, indentLevel int) {
	indent := ""
	for i := 0; i < indentLevel; i++ {
		indent += "  "
	}
	fmt.Printf("%s- Name: %s, Kind: %s\n", indent, symbol.Name, symbol.Kind)
	for _, child := range symbol.Children {
		printSymbol(r, child, indentLevel+1)
	}
}

// findSymbolByName recursively finds a symbol by name in a list of document symbols.
func findSymbolByName(symbols []*lsp.DocumentSymbol, name string) *lsp.DocumentSymbol {
	for _, s := range symbols {
		if s.Name == name {
			return s
		}
		if child := findSymbolByName(s.Children, name); child != nil {
			return child
		}
	}
	return nil
}

func TestJavaLSPSemanticFeatures(t *testing.T) {
	projectRoot := "../../../testdata/java/1_advanced" // New project root
	ctx := context.Background()

	openfile, wait := java.CheckRepo(projectRoot)
	l, s := java.GetDefaultLSP()
	lsp.RegisterProvider(uniast.Java, &JavaProvider{})

	lspClient, err := lsp.NewLSPClient(projectRoot, openfile, wait, lsp.ClientOptions{
		Server:   s,
		Language: l,
		Verbose:  true,
	})
	if err != nil {
		t.Fatalf("init lspclient failed = %v\n", err)
	}

	// --- Step 1 & 2: Connect and initialize the LSP client ---
	require.NoError(t, err, "Failed to initialize LSP client")
	// lspClient.SetVerbose(true)
	lspClient.InitFiles()

	// --- Step 3: Open all relevant documents to make them known to the LSP server ---
	animalFile := "../../../testdata/java/1_advanced/src/main/java/org/example/Animal.java"
	dogFile := "../../../testdata/java/1_advanced/src/main/java/org/example/Dog.java"
	catFile := "../../../testdata/java/1_advanced/src/main/java/org/example/Cat.java"

	animalURI := lsp.NewURI(animalFile)
	dogURI := lsp.NewURI(dogFile)
	catURI := lsp.NewURI(catFile)

	_, err = lspClient.DidOpen(ctx, animalURI)
	require.NoError(t, err, "textDocument/didOpen failed for Animal.java")
	_, err = lspClient.DidOpen(ctx, dogURI)
	require.NoError(t, err, "textDocument/didOpen failed for Dog.java")
	_, err = lspClient.DidOpen(ctx, catURI)
	require.NoError(t, err, "textDocument/didOpen failed for Cat.java")

	// Allow time for the LSP server to index the files before querying.
	time.Sleep(2 * time.Second)

	// --- Step 5: Test 'textDocument/implementation' to find implementations of Animal ---
	animalSymbolsMap, err := lspClient.DocumentSymbols(ctx, animalURI)
	require.NoError(t, err, "textDocument/documentSymbol request failed for Animal.java")
	require.NotEmpty(t, animalSymbolsMap, "Expected to find symbols in Animal.java")

	var animalSymbols []*lsp.DocumentSymbol
	for _, s := range animalSymbolsMap {
		animalSymbols = append(animalSymbols, s)
	}

	animalInterfaceSymbol := findSymbolByName(animalSymbols, "Animal")
	require.NotNil(t, animalInterfaceSymbol, "Could not find 'Animal' interface symbol")

	implementations, err := lspClient.Implementation(ctx, animalURI, animalInterfaceSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/implementation request failed")
	require.Len(t, implementations, 2, "Expected to find 2 implementations of Animal interface")

	fmt.Println("\n--- Found 2 implementations for interface 'Animal' ---")
	var foundDog, foundCat bool
	for _, impl := range implementations {
		if strings.HasSuffix(string(impl.URI), "Dog.java") {
			foundDog = true
		}
		if strings.HasSuffix(string(impl.URI), "Cat.java") {
			foundCat = true
		}
	}
	require.True(t, foundDog, "Did not find implementation in Dog.java")
	require.True(t, foundCat, "Did not find implementation in Cat.java")
	fmt.Println("---------------------------------------------------------")

	// --- Step 6: Test 'textDocument/definition' for a cross-file scenario ---
	// This part remains the same, as it verifies that we can still go from an implementation to the definition.
	// We will find the definition of `makeSound` from the `Dog` class implementation.

	// First, find the position of the 'makeSound' method in Dog.java using FileStructure
	dogSymbols2, err := lspClient.FileStructure(ctx, dogURI)
	require.NoError(t, err, "FileStructure request failed for Dog.java")
	makeSoundInDogSymbol := findSymbolByName(dogSymbols2, "makeSound()")
	require.NotNil(t, makeSoundInDogSymbol, "Could not find 'makeSound' method in 'Dog' class")

	// Use the position of 'makeSound' in 'Dog' to find its declaration. This is a semantic query.
	declarationResult, err := lspClient.Declaration(ctx, dogURI, makeSoundInDogSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/declaration request failed")
	require.NotEmpty(t, declarationResult, "Expected a declaration result")

	fmt.Println("\n--- Go to Declaration Result for Dog.makeSound() ---")
	declaration := declarationResult[0]
	fmt.Printf("Declaration found at: %s, Line: %d\n", declaration.URI, declaration.Range.Start.Line+1)

	// The declaration should be in Animal.java
	require.True(t, strings.HasSuffix(string(declaration.URI), "Animal.java"), "Declaration should be in Animal.java")
}

func TestJavaLSPInheritanceFeatures(t *testing.T) {
	projectRoot := "../../../testdata/java/2_inheritance"
	ctx := context.Background()

	openfile, wait := java.CheckRepo(projectRoot)
	l, s := java.GetDefaultLSP()
	lsp.RegisterProvider(uniast.Java, &JavaProvider{})

	lspClient, err := lsp.NewLSPClient(projectRoot, openfile, wait, lsp.ClientOptions{
		Server:   s,
		Language: l,
		Verbose:  false,
	})
	if err != nil {
		t.Fatalf("init lspclient failed = %v\n", err)
	}

	require.NoError(t, err, "Failed to initialize LSP client")
	// lspClient.SetVerbose(true)

	lspClient.InitFiles()

	shapeFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Shape.java"
	circleFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Circle.java"
	rectangleFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Rectangle.java"

	shapeURI := lsp.NewURI(shapeFile)
	circleURI := lsp.NewURI(circleFile)
	rectangleURI := lsp.NewURI(rectangleFile)

	_, err = lspClient.DidOpen(ctx, shapeURI)
	require.NoError(t, err, "textDocument/didOpen failed for Shape.java")
	_, err = lspClient.DidOpen(ctx, circleURI)
	require.NoError(t, err, "textDocument/didOpen failed for Circle.java")
	_, err = lspClient.DidOpen(ctx, rectangleURI)
	require.NoError(t, err, "textDocument/didOpen failed for Rectangle.java")

	time.Sleep(2 * time.Second)

	// --- Step 1: Test 'textDocument/references' for the abstract method ---
	shapeSymbols, err := lspClient.FileStructure(ctx, shapeURI)
	require.NoError(t, err, "FileStructure request failed for Shape.java")
	drawMethodSymbol := findSymbolByName(shapeSymbols, "draw()")
	require.NotNil(t, drawMethodSymbol, "Could not find 'draw' method in 'Shape' class")

	references, err := lspClient.References(ctx, drawMethodSymbol.Location)
	require.NoError(t, err, "textDocument/references request failed")
	require.Len(t, references, 3, "Expected to find 3 references to draw(), including the declaration")

	fmt.Println("\n--- Found 3 references for abstract method 'draw()' ---")

	var foundCircle, foundRectangle, foundShape bool
	for _, ref := range references {
		if strings.HasSuffix(string(ref.URI), "Circle.java") {
			foundCircle = true
		}
		if strings.HasSuffix(string(ref.URI), "Rectangle.java") {
			foundRectangle = true
		}
		if strings.HasSuffix(string(ref.URI), "Shape.java") {
			foundShape = true
		}
	}
	require.True(t, foundCircle, "Did not find reference in Circle.java")
	require.True(t, foundRectangle, "Did not find reference in Rectangle.java")
	require.True(t, foundShape, "Did not find reference in Shape.java")

	// --- Step 2: Test 'textDocument/definition' from implementation to abstract class ---
	circleSymbols, err := lspClient.FileStructure(ctx, circleURI)
	require.NoError(t, err, "FileStructure request failed for Circle.java")
	drawInCircleSymbol := findSymbolByName(circleSymbols, "draw()")
	require.NotNil(t, drawInCircleSymbol, "Could not find 'draw' method in 'Circle' class")

	declarationResult, err := lspClient.Declaration(ctx, circleURI, drawInCircleSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/declaration request failed")
	require.NotEmpty(t, declarationResult, "Expected a declaration result")
	declaration := declarationResult[0]
	require.True(t, strings.HasSuffix(string(declaration.URI), "Shape.java"), "Declaration should be in Shape.java")

	fmt.Println("\n--- Go to Declaration Result for Circle.draw() ---")
	fmt.Printf("Declaration found at: %s, Line: %d\n", declaration.URI, declaration.Range.Start.Line+1)

	// --- Step 3: Test 'textDocument/typeDefinition' on a class instance ---
	circleSymbolsForType, err := lspClient.FileStructure(ctx, circleURI)
	require.NoError(t, err, "FileStructure request failed for Circle.java")
	circleClassSymbol := findSymbolByName(circleSymbolsForType, "Circle")
	require.NotNil(t, circleClassSymbol, "Could not find 'Circle' class symbol")

	typeDefinitionResult, err := lspClient.TypeDefinition(ctx, circleURI, circleClassSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/typeDefinition request failed")
	require.NotEmpty(t, typeDefinitionResult, "Expected a type definition result")
	typeDefinition := typeDefinitionResult[0]
	require.True(t, strings.HasSuffix(string(typeDefinition.URI), "Circle.java"), "Type definition should be in Circle.java")

	fmt.Println("\n--- Go to Type Definition Result for Circle ---")
	fmt.Printf("Type Definition found at: %s, Line: %d\n", typeDefinition.URI, typeDefinition.Range.Start.Line+1)
}

func TestJavaLSPTypeHierarchy(t *testing.T) {
	projectRoot := "../../../testdata/java/2_inheritance"
	ctx := context.Background()

	openfile, wait := java.CheckRepo(projectRoot)
	l, s := java.GetDefaultLSP()
	lsp.RegisterProvider(uniast.Java, &JavaProvider{})

	lspClient, err := lsp.NewLSPClient(projectRoot, openfile, wait, lsp.ClientOptions{
		Server:   s,
		Language: l,
		Verbose:  false,
	})
	if err != nil {
		t.Fatalf("init lspclient failed = %v\n", err)
	}
	require.NoError(t, err, "Failed to initialize LSP client")
	// lspClient.SetVerbose(true)

	lspClient.InitFiles()

	shapeFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Shape.java"
	circleFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Circle.java"
	rectangleFile := "../../../testdata/java/2_inheritance/src/main/java/org/example/Rectangle.java"

	shapeURI := lsp.NewURI(shapeFile)
	circleURI := lsp.NewURI(circleFile)
	rectangleURI := lsp.NewURI(rectangleFile)

	_, err = lspClient.DidOpen(ctx, shapeURI)
	require.NoError(t, err, "textDocument/didOpen failed for Shape.java")
	_, err = lspClient.DidOpen(ctx, circleURI)
	require.NoError(t, err, "textDocument/didOpen failed for Circle.java")
	_, err = lspClient.DidOpen(ctx, rectangleURI)
	require.NoError(t, err, "textDocument/didOpen failed for Rectangle.java")

	time.Sleep(2 * time.Second)

	// --- Step 1: Test 'typeHierarchy/subtypes' for Shape ---
	shapeSymbols, err := lspClient.FileStructure(ctx, shapeURI)
	require.NoError(t, err, "FileStructure request failed for Shape.java")
	shapeSymbol := findSymbolByName(shapeSymbols, "Shape")
	require.NotNil(t, shapeSymbol, "Could not find 'Shape' class symbol")

	// Prepare the type hierarchy
	shapeItems, err := lspClient.PrepareTypeHierarchy(ctx, shapeURI, shapeSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/prepareTypeHierarchy request failed for Shape")
	require.Len(t, shapeItems, 1, "Expected one type hierarchy item for Shape")
	shapeItem := shapeItems[0]

	// Get subtypes
	subtypes, err := lspClient.TypeHierarchySubtypes(ctx, shapeItem)
	require.NoError(t, err, "typeHierarchy/subtypes request failed")
	require.Len(t, subtypes, 2, "Expected to find 2 subtypes of Shape")

	fmt.Println("\n--- Found 2 subtypes for class 'Shape' ---")
	var foundCircle, foundRectangle bool
	for _, child := range subtypes {
		if child.Name == "Circle" {
			foundCircle = true
		}
		if child.Name == "Rectangle" {
			foundRectangle = true
		}
	}
	require.True(t, foundCircle, "Did not find subtype Circle")
	require.True(t, foundRectangle, "Did not find subtype Rectangle")

	// --- Step 2: Test 'typeHierarchy/supertypes' for Circle ---
	circleSymbols, err := lspClient.FileStructure(ctx, circleURI)
	require.NoError(t, err, "FileStructure request failed for Circle.java")
	circleSymbol := findSymbolByName(circleSymbols, "Circle")
	require.NotNil(t, circleSymbol, "Could not find 'Circle' class symbol")

	// Prepare the type hierarchy
	circleItems, err := lspClient.PrepareTypeHierarchy(ctx, circleURI, circleSymbol.Location.Range.Start)
	require.NoError(t, err, "textDocument/prepareTypeHierarchy request failed for Circle")
	require.Len(t, circleItems, 1, "Expected one type hierarchy item for Circle")
	circleItem := circleItems[0]

	// Get supertypes
	supertypes, err := lspClient.TypeHierarchySupertypes(ctx, circleItem)
	require.NoError(t, err, "typeHierarchy/supertypes request failed")
	require.Len(t, supertypes, 1, "Expected to find 1 supertype of Circle")

	fmt.Println("\n--- Found 1 supertype for class 'Circle' ---")
	require.Equal(t, "Shape", supertypes[0].Name, "Supertype of Circle should be Shape")
}
