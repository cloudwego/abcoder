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

package parser

import (
	"container/list"
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/Knetic/govaluate"
	. "github.com/cloudwego/abcoder/lang/uniast"
	"golang.org/x/mod/modfile"
)

func shouldIgnoreDir(path string) bool {
	return strings.Contains(path, ".git") || strings.Contains(path, "vendor/") || strings.Contains(path, "kitex_gen") || strings.Contains(path, "hertz_gen")
}

func shouldIgnoreFile(path string) bool {
	return !strings.Contains(path, ".go") || strings.Contains(path, "_test.go")
}

type cache map[interface{}]bool

func (c cache) Visited(val interface{}) bool {
	ok := c[val]
	if !ok {
		c[val] = true
	}
	return ok
}

type cacheEntry struct {
	key   string
	value bool
}

// PackageCache 缓存 importPath 是否是 system package
type PackageCache struct {
	lock        sync.Mutex
	cache       map[string]*list.Element
	lru         *list.List
	lruCapacity int
}

func NewPackageCache(lruCapacity int) *PackageCache {
	return &PackageCache{
		cache:       make(map[string]*list.Element),
		lru:         list.New(),
		lruCapacity: lruCapacity,
	}
}

// get retrieves a value from the cache.
func (pc *PackageCache) get(key string) (bool, bool) {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	if elem, ok := pc.cache[key]; ok {
		pc.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry).value, true
	}
	return false, false
}

// set adds a value to the cache.
func (pc *PackageCache) set(key string, value bool) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	if elem, ok := pc.cache[key]; ok {
		pc.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	if pc.lru.Len() >= pc.lruCapacity {
		oldest := pc.lru.Back()
		if oldest != nil {
			pc.lru.Remove(oldest)
			delete(pc.cache, oldest.Value.(*cacheEntry).key)
		}
	}

	elem := pc.lru.PushFront(&cacheEntry{key: key, value: value})
	pc.cache[key] = elem
}

// IsStandardPackage 检查一个包是否为标准库，并使用内部缓存。
func (pc *PackageCache) IsStandardPackage(path string) bool {
	if isStd, found := pc.get(path); found {
		return isStd
	}

	pkg, err := build.Import(path, "", build.FindOnly)
	if err != nil {
		// Cannot find the package, assume it's not a standard package
		pc.set(path, false)
		return false
	}

	isStd := pkg.Goroot
	pc.set(path, isStd)
	return isStd
}

// stdlibCache 缓存 importPath 是否是 system package, 10000 个缓存
var stdlibCache = NewPackageCache(10000)

func isSysPkg(importPath string) bool {
	return stdlibCache.IsStandardPackage(importPath)
}

var (
	verReg = regexp.MustCompile(`/v\d+$`)
)

func getPackageAlias(importPath string) string {
	// Remove the version suffix if present (e.g., "/v2" or "/v10")

	basePath := verReg.ReplaceAllString(importPath, "")

	// Get the base name of the package
	alias := path.Base(basePath)

	// Replace any non-valid identifier characters with underscores
	if ps := strings.Split(alias, "-"); len(ps) > 1 {
		alias = ps[1]
	}

	return alias
}

func hasNoDeps(modFilePath string) bool {
	content, err := os.ReadFile(modFilePath)
	if err != nil {
		return false
	}

	modf, err := modfile.Parse(modFilePath, content, nil)
	if err != nil {
		return false
	}

	return len(modf.Require) == 0
}

func getModuleName(modFilePath string) (string, error) {
	content, err := os.ReadFile(modFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return modfile.ModulePath(content), nil
}

func isGoBuiltins(name string) bool {
	switch name {
	case "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover":
		return true
	case "string", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	case "interface{}", "any":
		return true
	default:
		return false
	}
}

func isPkgScope(scope *types.Scope) bool {
	return scope != nil && scope.Parent() == types.Universe
}

func getTypeKind(n ast.Expr) TypeKind {
	switch n.(type) {
	case *ast.StructType:
		return TypeKindStruct
	case *ast.InterfaceType:
		return TypeKindInterface
	default:
		return TypeKindTypedef
	}
}

func getNamedTypes(typ types.Type, visited map[types.Type]bool) (tys []types.Object, isPointer bool, isNamed bool) {
	if visited[typ] {
		return nil, false, false
	}

	visited[typ] = true

	switch t := typ.(type) {
	case *types.Pointer:
		isPointer = true
		var typs []types.Object
		typs, _, isNamed = getNamedTypes(t.Elem(), visited)
		tys = append(tys, typs...)
	case *types.Slice:
		typs, _, _ := getNamedTypes(t.Elem(), visited)
		tys = append(tys, typs...)
	case *types.Array:
		typs, _, _ := getNamedTypes(t.Elem(), visited)
		tys = append(tys, typs...)
	case *types.Chan:
		typs, _, _ := getNamedTypes(t.Elem(), visited)
		tys = append(tys, typs...)
	case *types.Tuple:
		for i := 0; i < t.Len(); i++ {
			typs, _, _ := getNamedTypes(t.At(i).Type(), visited)
			tys = append(tys, typs...)
		}
	case *types.Map:
		typs2, _, _ := getNamedTypes(t.Elem(), visited)
		typs1, _, _ := getNamedTypes(t.Key(), visited)
		tys = append(tys, typs1...)
		tys = append(tys, typs2...)
	case *types.Named:
		tys = append(tys, t.Obj())
		isNamed = true
		if targs := t.TypeArgs(); targs != nil {
			for i := 0; i < targs.Len(); i++ {
				typs, _, _ := getNamedTypes(targs.At(i), visited)
				tys = append(tys, typs...)
			}
		}
		if tparams := t.TypeParams(); tparams != nil {
			for i := 0; i < tparams.Len(); i++ {
				typs, _, _ := getNamedTypes(tparams.At(i), visited)
				tys = append(tys, typs...)
			}
		}
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			typs, _, _ := getNamedTypes(t.Field(i).Type(), visited)
			tys = append(tys, typs...)
		}
	case *types.Interface:
		for i := 0; i < t.NumEmbeddeds(); i++ {
			typs, _, _ := getNamedTypes(t.EmbeddedType(i), visited)
			tys = append(tys, typs...)
		}
		for i := 0; i < t.NumExplicitMethods(); i++ {
			typs, _, _ := getNamedTypes(t.ExplicitMethod(i).Type(), visited)
			tys = append(tys, typs...)
		}
	case *types.TypeParam:
		typs, _, _ := getNamedTypes(t.Constraint(), visited)
		tys = append(tys, typs...)
	case *types.Alias:
		var typs []types.Object
		typs, isPointer, isNamed = getNamedTypes(t.Rhs(), visited)
		tys = append(tys, typs...)
	case *types.Signature:
		for i := 0; i < t.Params().Len(); i++ {
			typs, _, _ := getNamedTypes(t.Params().At(i).Type(), visited)
			tys = append(tys, typs...)
		}
		for i := 0; i < t.Results().Len(); i++ {
			typs, _, _ := getNamedTypes(t.Results().At(i).Type(), visited)
			tys = append(tys, typs...)
		}
	}
	return
}

func parseExpr(expr string) (interface{}, error) {
	// Create a map of parameters to pass to the expression evaluator.
	parameters := map[string]interface{}{
		"iota": 0,
	}

	// Create the expression evaluator.
	eval, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, err
	}

	result, err := eval.Evaluate(parameters)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func newIdentity(mod, pkg, name string) Identity {
	return Identity{ModPath: mod, PkgPath: pkg, Name: name}
}

func isUpperCase(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func getCommitHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
