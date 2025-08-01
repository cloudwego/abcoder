/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package parser

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	. "github.com/cloudwego/abcoder/lang/uniast"
	"github.com/joyme123/thrift-ls/format"
	"github.com/joyme123/thrift-ls/lsp/cache"
	"github.com/joyme123/thrift-ls/lsp/lsputils"
	"github.com/joyme123/thrift-ls/lsp/memoize"
	"github.com/joyme123/thrift-ls/parser"
	"go.lsp.dev/uri"
)

var _ Parser = (*ThriftParser)(nil)

// ThriftParser holds the state and logic for parsing a repository of Thrift files into a UniAST structure.
type ThriftParser struct {
	rootDir   string     // Absolute path to the repository root.
	repo      Repository // The UniAST repository object being built.
	opts      Options    // Specific options for Thrift parsing.
	fileCache map[string][]byte
	fileAst   map[string]*parser.Document
	excludes  []*regexp.Regexp // Regular expressions for files/directories to exclude.

	parsedFiles      map[string]bool
	modName          string
	includeRelations map[string]map[string]string
	fileToNamespace  map[string]string

	initialFileChanges []*cache.FileChange // A list of initial file changes to build the AST.
}

// NewParser creates and initializes a new ThriftParser.
func NewParser(rootDir string, opts Options) (*ThriftParser, error) {
	if opts.TargetLanguage == "" {
		return nil, fmt.Errorf("TargetLanguage option is required")
	}

	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for rootDir: %w", err)
	}

	p := &ThriftParser{
		rootDir:          absRootDir,
		repo:             NewRepository(rootDir),
		opts:             opts,
		fileCache:        make(map[string][]byte),
		parsedFiles:      make(map[string]bool),
		fileAst:          make(map[string]*parser.Document),
		modName:          "current",
		includeRelations: make(map[string]map[string]string),
		fileToNamespace:  make(map[string]string),
	}
	p.repo.Modules["current"] = NewModule("current", ".", Thrift)

	for _, ex := range opts.Excludes {
		r, err := regexp.Compile(ex)
		if err != nil {
			log.Error("Warning: failed to compile exclude pattern '%s': %v\n", ex, err)
		} else {
			p.excludes = append(p.excludes, r)
		}
	}

	if err := p.preScanThriftFiles(); err != nil {
		return nil, fmt.Errorf("failed during pre-scan: %w", err)
	}

	return p, nil
}

// preScanThriftFiles walks the root directory to find all .thrift files,
// caches their content, and performs an initial parse.
func (p *ThriftParser) preScanThriftFiles() error {
	var fileChanges []*cache.FileChange

	err := filepath.Walk(p.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		for _, r := range p.excludes {
			if r.MatchString(path) {
				return filepath.SkipDir
			}
		}

		if !strings.HasSuffix(path, ".thrift") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Quick scan for namespaces and includes without a full parse.
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "namespace") {
				parts := strings.Fields(line)
				// e.g., "namespace go abc.def"
				if len(parts) == 3 && parts[1] == p.opts.TargetLanguage {
					p.fileToNamespace[path] = parts[2]
				}
			} else if strings.HasPrefix(line, "include") {
				parts := strings.Fields(line)
				if len(parts) == 2 {
					includePathRaw := strings.Trim(parts[1], `"'`)
					absIncludePath := filepath.Join(filepath.Dir(path), includePathRaw)
					alias := strings.TrimSuffix(filepath.Base(includePathRaw), ".thrift")
					if p.includeRelations[path] == nil {
						p.includeRelations[path] = make(map[string]string)
					}
					p.includeRelations[path][alias] = absIncludePath
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		absPath, _ := filepath.Abs(path)
		var finalAST *parser.Document

		// Initial parse to get the AST.
		initialAST, err := parser.Parse(absPath, content)
		if err != nil {
			log.Error("Initial parse failed for %s: %v", absPath, err)
			finalAST = nil // Continue even if parsing fails.
		} else {
			finalAST = initialAST.(*parser.Document)
		}

		if !p.opts.CollectComment && finalAST != nil {
			// Remove comments and re-parse to get clean offsets.
			removeAllComments(finalAST)
			contentString, err := format.FormatDocument(finalAST)
			if err != nil {
				return err // Formatting failure is a critical error.
			}
			content = []byte(contentString)

			reParsedAST, err := parser.Parse(absPath, content)
			if err != nil {
				log.Error("Re-parse after comment removal failed for %s: %v", absPath, err)
				// If re-parse fails, we set the AST to nil.
				// Falling back to the commented AST would complicate offset logic.
				finalAST = nil
			} else {
				finalAST = reParsedAST.(*parser.Document)
			}
		}

		uriFile := uri.File(absPath)
		p.fileAst[uriFile.Filename()] = finalAST
		p.fileCache[uriFile.Filename()] = content
		fileChanges = append(fileChanges, &cache.FileChange{
			URI:     uriFile,
			Content: content,
			From:    cache.FileChangeTypeDidOpen,
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk thrift files: %w", err)
	}
	p.initialFileChanges = fileChanges
	return nil
}

// ParseRepo parses the entire repository and builds the UniAST graph.
func (p *ThriftParser) ParseRepo() (Repository, error) {
	for _, fc := range p.initialFileChanges {
		if err := p.collectEntitiesFromURI(&p.repo, fc.URI); err != nil {
			log.Error("Error processing file '%s': %v", fc.URI.Filename(), err)
			return p.repo, err
		}
	}

	if err := p.repo.BuildGraph(); err != nil {
		return p.repo, fmt.Errorf("failed to build UniAST graph: %w", err)
	}

	return p.repo, nil
}

// findNamespace finds the namespace declaration for the target language.
func (p *ThriftParser) findNamespace(doc *parser.Document) PkgPath {
	for _, ns := range doc.Namespaces {
		if ns.Language.Name.Text == p.opts.TargetLanguage {
			return ns.Name.Name.Text
		}
	}
	// Fallback to wildcard namespace if available.
	for _, ns := range doc.Namespaces {
		if ns.Language.Name.Text == "*" {
			return ns.Name.Name.Text
		}
	}
	return ""
}

// collectServices extracts service and function definitions from a Thrift document.
func (p *ThriftParser) collectServices(doc *parser.Document, pkg *Package, relFilePath string, fileURI uri.URI) error {
	content := p.fileCache[fileURI.Filename()]

	var addDependenciesToSlice func(dependencies *[]Dependency, ft *parser.FieldType)
	addDependenciesToSlice = func(dependencies *[]Dependency, ft *parser.FieldType) {
		if ft == nil {
			return
		}

		identity, err := p.fieldTypeToIdentity(fileURI, doc, ft)
		if err != nil {
			log.Error("Failed to resolve type identity for '%s' in '%s': %v", ft.TypeName.Name, fileURI.Filename(), err)
			return
		}
		sp, ep := p.getRealFieldTypePositions(ft)
		// Only add dependencies that are custom types (i.e., have a PkgPath).
		if identity.PkgPath != "" {
			dep := NewDependency(*identity, FileLine{
				File:        relFilePath,
				Line:        p.getRealFieldTypeLine(ft),
				StartOffset: sp.Offset,
				EndOffset:   ep.Offset,
			})
			*dependencies = InsertDependency(*dependencies, dep)
		}

		// Recursively add dependencies for container types.
		if ft.KeyType != nil {
			addDependenciesToSlice(dependencies, ft.KeyType)
		}
		if ft.ValueType != nil {
			addDependenciesToSlice(dependencies, ft.ValueType)
		}
	}

	for _, service := range doc.Services {
		serviceIdentity := NewIdentity(p.modName, pkg.PkgPath, service.Name.Name.Text)
		for _, function := range service.Functions {
			funcName := fmt.Sprintf("%s.%s", service.Name.Name.Text, function.Name.Name.Text)
			funcIdentity := NewIdentity(p.modName, pkg.PkgPath, funcName)

			signature, err := p.getFuncSignature(function, content)
			if err != nil {
				log.Error("Failed to get signature for function '%s': %v", funcName, err)
			}

			uniFunc := &Function{
				Exported:          true,
				IsMethod:          true,
				IsInterfaceMethod: false,
				Identity:          funcIdentity,
				FileLine: FileLine{
					File:        relFilePath,
					Line:        p.getRealFuncStartLine(function),
					StartOffset: p.getFuncStartOffset(function, false),
					EndOffset:   p.getRealFuncEndOffset(function, false),
				},
				Content:   format.MustFormatService(service),
				Signature: signature,
				Receiver: &Receiver{
					IsPointer: false,
					Type:      serviceIdentity,
				},
				Params:  make([]Dependency, 0),
				Results: make([]Dependency, 0),
			}

			for _, arg := range function.Arguments {
				addDependenciesToSlice(&uniFunc.Params, arg.FieldType)
			}

			if function.Oneway == nil && function.FunctionType != nil {
				addDependenciesToSlice(&uniFunc.Results, function.FunctionType)
			}

			pkg.Functions[funcName] = uniFunc
		}
	}
	return nil
}

// fieldTypeToIdentity converts a Thrift FieldType to a UniAST Identity.
func (p *ThriftParser) fieldTypeToIdentity(currentFileURI uri.URI, currentDoc *parser.Document, fieldType *parser.FieldType) (*Identity, error) {
	if fieldType == nil || fieldType.TypeName == nil {
		return &Identity{ModPath: p.modName, PkgPath: p.findNamespace(currentDoc), Name: "unknown"}, nil
	}
	return p.resolveTypeIdentity(currentFileURI, currentDoc, fieldType.TypeName.Name)
}

// resolveTypeIdentity resolves a type name to its full UniAST Identity, handling includes and namespaces.
func (p *ThriftParser) resolveTypeIdentity(currentFileURI uri.URI, currentDoc *parser.Document, typeName string) (*Identity, error) {
	baseTypes := map[string]bool{
		"bool": true, "byte": true, "i8": true, "i16": true, "i32": true, "i64": true,
		"double": true, "string": true, "binary": true, "uuid": true,
		"list": true, "set": true, "map": true,
	}

	if baseTypes[typeName] {
		// Base types have no module or package path.
		return &Identity{Name: typeName}, nil
	}

	alias := ""
	typeNamePart := typeName
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		alias = parts[0]
		typeNamePart = parts[1]
	}

	var targetDoc *parser.Document
	var targetFileURI uri.URI

	if alias == "" {
		// Type is defined in the current file.
		targetDoc = currentDoc
		targetFileURI = currentFileURI
	} else {
		// Type is imported from another file.
		includePath, found := "", false
		for _, inc := range currentDoc.Includes {
			incAlias := strings.TrimSuffix(filepath.Base(inc.Path.Value.Text), ".thrift")
			if incAlias == alias {
				includePath = inc.Path.Value.Text
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("include declaration with alias '%s' not found in file '%s'", alias, currentFileURI.Filename())
		}

		targetFileURI = lsputils.IncludeURI(currentFileURI, includePath)
		parsedFile := p.fileAst[targetFileURI.Filename()]
		targetDoc = parsedFile
	}

	pkgPath := p.findNamespace(targetDoc)
	if pkgPath == "" {
		log.Error("No suitable namespace found for target language '%s' in file '%s'", p.opts.TargetLanguage, targetFileURI.Filename())
		return &Identity{ModPath: p.modName, Name: typeNamePart}, fmt.Errorf("namespace not found in %s", targetFileURI.Filename())
	}

	identity := NewIdentity(p.modName, pkgPath, typeNamePart)
	return &identity, nil
}

// collectConsts extracts const definitions from a Thrift document.
func (p *ThriftParser) collectConsts(doc *parser.Document, content []byte, pkg *Package, relFilePath string) {
	vars, err := p.collectThriftVars(doc, content, p.modName, relFilePath)
	if err == nil {
		for k, v := range vars {
			pkg.Vars[k] = v
		}
	}
}

// collectTypes extracts type definitions (structs, enums, etc.) from a Thrift document.
func (p *ThriftParser) collectTypes(doc *parser.Document, pkg *Package, relFilePath string, fileURI uri.URI) {
	types, err := p.collectThriftTypes(doc, pkg.PkgPath, p.modName, relFilePath, fileURI)
	if err == nil {
		for k, v := range types {
			pkg.Types[k] = v
		}
	}
}

// collectThriftTypes is a helper that performs the actual extraction of various type definitions.
func (p *ThriftParser) collectThriftTypes(doc *parser.Document, pkgPath PkgPath, modPath ModPath, filePath string, fileURI uri.URI) (map[string]*Type, error) {
	types := make(map[string]*Type)

	for _, s := range doc.Structs {
		name := s.Identifier.Name.Text
		sp, ep := p.getRealStructPositions(s)
		uniType := &Type{
			Exported: true,
			TypeKind: TypeKindStruct,
			Identity: NewIdentity(modPath, pkgPath, name),
			FileLine: newFileLine(filePath, p.getRealStructLine(s), sp, ep),
			Content:  format.MustFormatStruct(s),
		}
		p.processStructLike(s.Fields, uniType, fileURI, doc)
		types[name] = uniType
	}

	for _, e := range doc.Exceptions {
		name := e.Name.Name.Text
		sp, ep := p.getRealExceptionPositions(e)
		uniType := &Type{
			Exported: true,
			TypeKind: TypeKindStruct, // Exceptions are structurally similar to structs.
			Identity: NewIdentity(modPath, pkgPath, name),
			FileLine: newFileLine(filePath, p.getRealExceptionLine(e), sp, ep),
			Content:  format.MustFormatException(e),
		}
		p.processStructLike(e.Fields, uniType, fileURI, doc)
		types[name] = uniType
	}

	for _, u := range doc.Unions {
		name := u.Name.Name.Text
		sp, ep := p.getRealUnionPositions(u)
		uniType := &Type{
			Exported: true,
			TypeKind: TypeKindStruct, // Unions are also structurally similar to structs.
			Identity: NewIdentity(modPath, pkgPath, name),
			FileLine: newFileLine(filePath, p.getRealUnionLine(u), sp, ep),
			Content:  format.MustFormatUnion(u),
		}
		p.processStructLike(u.Fields, uniType, fileURI, doc)
		types[name] = uniType
	}

	for _, e := range doc.Enums {
		name := e.Name.Name.Text
		sp, ep := p.getRealEnumPositions(e)
		types[name] = &Type{
			Exported: true,
			TypeKind: TypeKindEnum,
			Identity: NewIdentity(modPath, pkgPath, name),
			FileLine: newFileLine(filePath, p.getRealEnumLine(e), sp, ep),
			Content:  format.MustFormatEnum(e),
		}
	}

	for _, t := range doc.Typedefs {
		name := t.Alias.Name.Text
		originalTypeIdentity, err := p.fieldTypeToIdentity(fileURI, doc, t.T)
		if err != nil {
			log.Error("Failed to resolve typedef for '%s': %v", name, err)
			continue
		}
		sp, ep := p.getRealTypedefPositions(t)
		dep := NewDependency(*originalTypeIdentity, FileLine{
			File:        filePath,
			Line:        p.getRealTypedefLine(t),
			StartOffset: sp.Offset,
			EndOffset:   ep.Offset,
		})
		types[name] = &Type{
			Exported:  true,
			TypeKind:  TypeKindTypedef,
			Identity:  NewIdentity(modPath, pkgPath, name),
			FileLine:  newFileLine(filePath, p.getRealTypedefLine(t), sp, ep),
			Content:   format.MustFormatTypedef(t),
			SubStruct: []Dependency{dep},
		}
	}

	for _, s := range doc.Services {
		name := s.Name.Name.Text
		sp, ep := p.getRealServicePositions(s)
		uniType := &Type{
			Exported: true,
			TypeKind: TypeKindInterface,
			Identity: NewIdentity(modPath, pkgPath, name),
			FileLine: newFileLine(filePath, p.getRealServiceLine(s), sp, ep),
			Content:  format.MustFormatService(s),
			Methods:  make(map[string]Identity),
		}
		for _, f := range s.Functions {
			methodName := f.Name.Name.Text
			methodIdentity := NewIdentity(modPath, pkgPath, fmt.Sprintf("%s.%s", name, methodName))
			uniType.Methods[methodName] = methodIdentity
		}
		types[name] = uniType
	}

	return types, nil
}

// collectEntitiesFromURI orchestrates the collection of all entities from a single file URI.
func (p *ThriftParser) collectEntitiesFromURI(repo *Repository, fileURI uri.URI) error {
	content := p.fileCache[fileURI.Filename()]
	document := p.fileAst[fileURI.Filename()]
	if document == nil {
		log.Error("AST for file '%s' not found in cache.", fileURI.Filename())
		return nil
	}

	relFilePath, _ := filepath.Rel(p.rootDir, fileURI.Filename())
	module := repo.Modules["current"]

	// Process file-level information (Imports, Package).
	uniastFile := NewFile(relFilePath)
	for _, include := range document.Includes {
		pathValue := include.Path.Value.Text
		uniastFile.Imports = append(uniastFile.Imports, NewImport(nil, fmt.Sprintf(`"%s"`, pathValue)))
	}
	namespace := p.findNamespace(document)
	uniastFile.Package = namespace
	module.Files[relFilePath] = uniastFile

	if namespace == "" {
		log.Info("No namespace found for language %s in file %s, skipping entity collection.", p.opts.TargetLanguage, relFilePath)
		return nil
	}

	if module.Packages[namespace] == nil {
		module.Packages[namespace] = NewPackage(namespace)
	}
	uniastPackage := module.Packages[namespace]

	if err := p.collectServices(document, uniastPackage, relFilePath, fileURI); err != nil {
		return err
	}
	p.collectTypes(document, uniastPackage, relFilePath, fileURI)
	p.collectConsts(document, content, uniastPackage, relFilePath)

	return nil
}

// ParseNode parses a single node and its direct dependencies from the repository.
func (p *ThriftParser) ParseNode(pkgPath, name string) (Repository, error) {
	outRepo := NewRepository(p.repo.Name)
	outRepo.Modules["current"] = NewModule("current", ".", Thrift)

	// Helper function to copy a node from the fully parsed p.repo to the outRepo.
	addNode := func(id Identity) {
		if outRepo.Modules["current"].Packages[id.PkgPath] == nil {
			outRepo.Modules["current"].Packages[id.PkgPath] = NewPackage(id.PkgPath)
		}
		outPkg := outRepo.Modules["current"].Packages[id.PkgPath]

		// Find the original node in the complete repository AST (p.repo) and copy it.
		sourcePkg := p.repo.Modules["current"].Packages[id.PkgPath]
		if sourcePkg == nil {
			return // The dependent package does not exist in the full AST, skip.
		}

		if fn, ok := sourcePkg.Functions[id.Name]; ok {
			outPkg.Functions[id.Name] = fn
		} else if t, ok := sourcePkg.Types[id.Name]; ok {
			outPkg.Types[id.Name] = t
		} else if v, ok := sourcePkg.Vars[id.Name]; ok {
			outPkg.Vars[id.Name] = v
		}
	}

	// Find the target node in the complete repository AST.
	pkg := p.repo.Modules["current"].Packages[pkgPath]
	if pkg == nil {
		return outRepo, fmt.Errorf("package '%s' not found in repository", pkgPath)
	}

	var targetIdentity *Identity
	nodeFound := false

	if fn, ok := pkg.Functions[name]; ok {
		targetIdentity = &fn.Identity
		nodeFound = true
	} else if t, ok := pkg.Types[name]; ok {
		targetIdentity = &t.Identity
		nodeFound = true
	} else if v, ok := pkg.Vars[name]; ok {
		targetIdentity = &v.Identity
		nodeFound = true
	}

	if !nodeFound {
		return outRepo, fmt.Errorf("node '%s' not found in package '%s'", name, pkgPath)
	}

	// Add the target node and its dependencies to the output repository.
	addNode(*targetIdentity)
	graphNode := p.repo.GetNode(*targetIdentity)
	if graphNode != nil {
		for _, relation := range graphNode.Dependencies {
			addNode(relation.Identity)
		}
	}

	if err := outRepo.BuildGraph(); err != nil {
		return outRepo, fmt.Errorf("failed to build UniAST graph for node '%s': %w", name, err)
	}

	return outRepo, nil
}

// ParsePackage parses all files belonging to a specific package path.
func (p *ThriftParser) ParsePackage(pkgPath PkgPath) (Repository, error) {
	outRepo := NewRepository(p.repo.Name)
	// FIX: Initialize the "current" module to prevent panic in collectEntitiesFromURI.
	outRepo.Modules["current"] = NewModule("current", ".", Thrift)

	found := false
	for file, namespace := range p.fileToNamespace {
		if namespace == pkgPath {
			found = true
			fileURI := uri.File(file)
			if err := p.collectEntitiesFromURI(&outRepo, fileURI); err != nil {
				log.Error("Error processing file '%s' for package '%s': %v", file, pkgPath, err)
			}
		}
	}

	if !found {
		return outRepo, fmt.Errorf("package not found: %s", pkgPath)
	}

	if err := outRepo.BuildGraph(); err != nil {
		return outRepo, fmt.Errorf("failed to build UniAST graph for package '%s': %w", pkgPath, err)
	}

	return outRepo, nil
}

// buildSnapshot initializes a thrift-ls snapshot for parsing. (Currently not used in the main flow but useful for LSP-based approaches)
func (p *ThriftParser) buildSnapshot(fileChanges []*cache.FileChange) (*cache.Snapshot, error) {
	if len(fileChanges) == 0 {
		return nil, fmt.Errorf("no .thrift files found to build snapshot")
	}

	store := &memoize.Store{}
	c := cache.New(store)
	fs := cache.NewOverlayFS(c)

	if err := fs.Update(context.TODO(), fileChanges); err != nil {
		return nil, fmt.Errorf("failed to update overlay FS: %w", err)
	}

	folderURI := uri.File(p.rootDir)
	view := cache.NewView(p.modName, folderURI, fs, store)
	ss := cache.NewSnapshot(view, store)

	for _, f := range fileChanges {
		document, err := ss.Parse(context.TODO(), f.URI)
		if err != nil {
			log.Error("Warning: error parsing file '%s': %v\n", f.URI.Filename(), err)
		}
		p.fileAst[f.URI.Filename()] = document.AST()
	}

	return ss, nil
}

// toIdentity converts a thrift-ls FieldType node to a UniAST Identity.
// This is a simplified implementation. A more robust version would need to handle
// alias resolution from 'include' statements to correctly determine the PkgPath for types like `shared.User`.
func toIdentity(fieldType *parser.FieldType, currentPkg PkgPath, modPath ModPath) *Identity {
	if fieldType == nil || fieldType.TypeName == nil {
		return &Identity{Name: "unknown"}
	}

	typeName := fieldType.TypeName.Name

	// Base Thrift types do not have a ModPath or PkgPath.
	baseTypes := map[string]bool{
		"bool": true, "byte": true, "i8": true, "i16": true, "i32": true, "i64": true,
		"double": true, "string": true, "binary": true, "uuid": true,
	}
	if baseTypes[typeName] {
		return &Identity{Name: typeName}
	}

	// Container types themselves are keywords; their dependencies are their inner types.
	// This function only identifies the container type itself.
	containerTypes := map[string]bool{"list": true, "set": true, "map": true}
	if containerTypes[typeName] {
		return &Identity{Name: typeName}
	}

	// For this simplified version, assume all other types are within the current package.
	return &Identity{
		ModPath: modPath,
		PkgPath: currentPkg,
		Name:    typeName,
	}
}

// collectThriftVars extracts all 'const' definitions from a document.
func (p *ThriftParser) collectThriftVars(doc *parser.Document, source []byte, modPath ModPath, filePath string) (map[string]*Var, error) {
	vars := make(map[string]*Var)

	pkgPath := p.findNamespace(doc)
	if pkgPath == "" {
		return nil, fmt.Errorf("no suitable namespace found for language '%s' in file '%s'", p.opts.TargetLanguage, doc.Filename)
	}

	for _, c := range doc.Consts {
		constName := c.Name.Name.Text
		sp, ep := p.getRealConstPositions(c)
		content := p.getRealContent(source, sp.Offset, ep.Offset)

		uniVar := &Var{
			IsExported: true, // Thrift consts are public by default.
			IsConst:    true,
			IsPointer:  false,
			Identity: Identity{
				ModPath: modPath,
				PkgPath: pkgPath,
				Name:    constName,
			},
			FileLine: newFileLine(filePath, p.getRealConstLine(c), sp, ep),
			Type:     toIdentity(c.ConstType, pkgPath, modPath),
			Content:  content,
		}

		// Handle dependencies on other enums or constants, e.g., `const MyStatus s = MyStatus.OK`.
		if c.Value.TypeName == "identifier" {
			if identVal, ok := c.Value.Value.(string); ok {
				// This is a simplified analysis. The dependency is on the `MyStatus.OK` Var.
				// We create an identity for it assuming it's in the same package.
				if strings.Contains(identVal, ".") {
					depIdentity := Identity{
						ModPath: modPath,
						PkgPath: pkgPath,
						Name:    identVal, // The dependency name is the full identifier, e.g., 'MyStatus.OK'.
					}
					uniVar.Dependencies = append(uniVar.Dependencies, Dependency{Identity: depIdentity})
				}
			}
		}

		vars[constName] = uniVar
	}

	return vars, nil
}

// processStructLike processes types with fields, such as struct, exception, and union,
// to find and add their field type dependencies.
func (p *ThriftParser) processStructLike(
	fields []*parser.Field,
	uniType *Type,
	currentFileURI uri.URI,
	currentDoc *parser.Document,
) {
	var addDependencies func(ft *parser.FieldType)
	addDependencies = func(ft *parser.FieldType) {
		if ft == nil {
			return
		}

		// Parse the main type (or a container type like 'list').
		identity, err := p.fieldTypeToIdentity(currentFileURI, currentDoc, ft)
		if err != nil {
			log.Error("Failed to resolve type identity for '%s' in '%s': %v", ft.TypeName.Name, currentFileURI.Filename(), err)
			return
		}

		// Only add dependencies for custom types that have a package path.
		if identity.PkgPath != "" {
			dep := NewDependency(*identity, FileLine{
				File:        uniType.File,
				Line:        ft.Location.StartPos.Line,
				StartOffset: ft.Location.StartPos.Offset,
				EndOffset:   ft.Location.EndPos.Offset,
			})
			uniType.SubStruct = InsertDependency(uniType.SubStruct, dep)
		}

		// Recursively add dependencies for container key/value types.
		if ft.KeyType != nil {
			addDependencies(ft.KeyType)
		}
		if ft.ValueType != nil {
			addDependencies(ft.ValueType)
		}
	}

	for _, field := range fields {
		addDependencies(field.FieldType)
	}
}
