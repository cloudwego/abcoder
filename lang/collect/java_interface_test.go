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

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	javaipc "github.com/cloudwego/abcoder/lang/java/ipc"
	javapb "github.com/cloudwego/abcoder/lang/java/pb"
	"github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// TestJavaIPC_InterfaceKindAndImplements drives the Java parser → universal AST
// pipeline for the two regressions:
//  1. Interface declarations must be exported with TypeKind == "interface".
//  2. A class that "implements I" must have I in Type.Implements (and not be
//     a plain SubStruct dependency).
//
// We hand-build the javaipc.Converter so the test does not require the real
// java parser binary; it still exercises ScannerByJavaIPC + Export end-to-end
// against the real fixture source files under testdata/java/5_interface_impl.
func TestJavaIPC_InterfaceKindAndImplements(t *testing.T) {
	repo := fixtureRepo(t)
	conv := buildInterfaceFixtureConverter(repo)

	cli := &lsp.LSPClient{ClientOptions: lsp.ClientOptions{Language: uniast.Java}}
	c := NewCollector(repo, cli)
	c.Language = uniast.Java
	c.NeedStdSymbol = true
	c.UseJavaIPC(conv)

	if _, err := c.ScannerByJavaIPC(context.Background()); err != nil {
		t.Fatalf("ScannerByJavaIPC failed: %v", err)
	}
	rep, err := c.Export(context.Background())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	types := collectExportedTypes(rep)

	animal, ok := types["Animal"]
	if !ok {
		t.Fatalf("Animal type not exported; got types: %v", typeNames(types))
	}
	if animal.TypeKind != uniast.TypeKindInterface {
		t.Errorf("Animal.TypeKind = %q, want %q", animal.TypeKind, uniast.TypeKindInterface)
	}

	swimmer, ok := types["Swimmer"]
	if !ok {
		t.Fatalf("Swimmer type not exported; got types: %v", typeNames(types))
	}
	if swimmer.TypeKind != uniast.TypeKindInterface {
		t.Errorf("Swimmer.TypeKind = %q, want %q", swimmer.TypeKind, uniast.TypeKindInterface)
	}

	dog, ok := types["Dog"]
	if !ok {
		t.Fatalf("Dog type not exported; got types: %v", typeNames(types))
	}
	if dog.TypeKind != uniast.TypeKindStruct {
		t.Errorf("Dog.TypeKind = %q, want %q", dog.TypeKind, uniast.TypeKindStruct)
	}
	if !containsIdentityName(dog.Implements, "Animal") {
		t.Errorf("Dog.Implements does not contain Animal; got %v", identityNames(dog.Implements))
	}
	if containsDependencyName(dog.SubStruct, "Animal") {
		t.Errorf("Dog.SubStruct should not duplicate the Animal implements relation; got %v",
			dependencyNames(dog.SubStruct))
	}

	fish, ok := types["Fish"]
	if !ok {
		t.Fatalf("Fish type not exported; got types: %v", typeNames(types))
	}
	if !containsIdentityName(fish.Implements, "Animal") {
		t.Errorf("Fish.Implements missing Animal; got %v", identityNames(fish.Implements))
	}
	if !containsIdentityName(fish.Implements, "Swimmer") {
		t.Errorf("Fish.Implements missing Swimmer; got %v", identityNames(fish.Implements))
	}
}

// fixtureRepo returns the absolute path to testdata/java/5_interface_impl.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "java", "5_interface_impl")
}

func buildInterfaceFixtureConverter(repo string) *javaipc.Converter {
	conv := javaipc.NewConverter(repo, "test-mod")

	srcDir := filepath.Join(repo, "src", "main", "java", "org", "example")
	mk := func(fqcn, fname string, kind javapb.ClassType, startLine, endLine int32, implements []string) *javapb.ClassInfo {
		return &javapb.ClassInfo{
			ClassName:       fqcn,
			PackageName:     "org.example",
			FilePath:        filepath.Join(srcDir, fname),
			ClassType:       kind,
			ImplementsTypes: implements,
			StartLine:       startLine,
			StartColumn:     1,
			EndLine:         endLine,
			EndColumn:       2,
			Source:          &javapb.SourceInfo{Type: javapb.SourceType_SOURCE_TYPE_LOCAL},
		}
	}
	conv.LocalClassCache["org.example.Animal"] =
		mk("org.example.Animal", "Animal.java", javapb.ClassType_CLASS_TYPE_INTERFACE, 3, 6, nil)
	conv.LocalClassCache["org.example.Swimmer"] =
		mk("org.example.Swimmer", "Swimmer.java", javapb.ClassType_CLASS_TYPE_INTERFACE, 3, 5, nil)
	conv.LocalClassCache["org.example.Dog"] =
		mk("org.example.Dog", "Dog.java", javapb.ClassType_CLASS_TYPE_CLASS, 3, 19,
			[]string{"org.example.Animal"})
	conv.LocalClassCache["org.example.Fish"] =
		mk("org.example.Fish", "Fish.java", javapb.ClassType_CLASS_TYPE_CLASS, 3, 23,
			[]string{"org.example.Animal", "org.example.Swimmer"})
	return conv
}

func collectExportedTypes(rep *uniast.Repository) map[string]*uniast.Type {
	out := map[string]*uniast.Type{}
	for _, mod := range rep.Modules {
		for _, pkg := range mod.Packages {
			for _, ty := range pkg.Types {
				out[ty.Identity.Name] = ty
			}
		}
	}
	return out
}

func typeNames(m map[string]*uniast.Type) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func containsIdentityName(ids []uniast.Identity, name string) bool {
	for _, id := range ids {
		if id.Name == name {
			return true
		}
	}
	return false
}

func identityNames(ids []uniast.Identity) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Name)
	}
	return out
}

func containsDependencyName(deps []uniast.Dependency, name string) bool {
	for _, d := range deps {
		if d.Identity.Name == name {
			return true
		}
	}
	return false
}

func dependencyNames(deps []uniast.Dependency) []string {
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		out = append(out, d.Identity.Name)
	}
	return out
}
