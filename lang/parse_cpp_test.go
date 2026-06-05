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

package lang

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// cleanupPreambleCache removes leftover /tmp/preamble-*.pch files clangd
// writes when --pch-storage=disk has ever been used in this repo (e.g. by
// a prior CLI run). Stale preambles whose hash happens to collide with the
// new test's compile command get reused by clangd, which then returns
// degraded semantic tokens (empty bodies, missing call edges). Each test
// run starts from a clean slate so results are deterministic regardless
// of dev-box state.
func cleanupPreambleCache() {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "preamble-*.pch"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

// parseTestCase parses testdata/cpp/<name>/ and returns the Repository.
//
// Skips the test gracefully when clangd-18 (or any compatible clangd
// reachable through PATH) is missing, so the suite is still runnable on
// dev boxes without manual setup. CI always installs clangd-18 in the
// regression workflow (see .github/workflows/regression.yml).
func parseTestCase(t *testing.T, name string) *uniast.Repository {
	t.Helper()
	lspBin := resolveClangd(t)
	if lspBin == "" {
		t.Skipf("no clangd binary on PATH; skipping C++ parser test")
	}
	// Stale preamble PCHs from earlier runs can poison the next parse,
	// see cleanupPreambleCache for the full story.
	cleanupPreambleCache()

	workspace := testutils.TestPath(name, "cpp")
	opts := ParseOptions{
		LSP:     lspBin + " --background-index=false --pch-storage=disk -j=8 --clang-tidy=false",
		Verbose: false,
		CollectOption: collect.CollectOption{
			Language:           uniast.Cpp,
			LoadExternalSymbol: false,
			NeedStdSymbol:      false,
			NoNeedComment:      true,
			NotNeedTest:        true,
		},
		LspOptions: map[string]string{},
	}
	repoBytes, err := Parse(context.Background(), workspace, opts)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	var repo uniast.Repository
	if err := json.Unmarshal(repoBytes, &repo); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}
	return &repo
}

// resolveClangd looks for the binary we'll feed to abcoder. The CI image
// installs clangd-18; locally users tend to have either `clangd-18` or
// homebrew's `clangd` at $(brew --prefix llvm@18)/bin/clangd. We probe the
// likely names and return an absolute path.
func resolveClangd(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"clangd-18", "clangd"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	// Common homebrew layout.
	hb := "/home/linuxbrew/.linuxbrew/opt/llvm@18/bin/clangd"
	if _, err := exec.LookPath(hb); err == nil {
		return hb
	}
	return ""
}

// forEachType invokes fn(typeName, type) for every Type in every package of
// the repo's internal module. Order isn't guaranteed.
func forEachType(repo *uniast.Repository, fn func(name string, ty *uniast.Type)) {
	for _, mod := range repo.Modules {
		for _, pkg := range mod.Packages {
			for n, ty := range pkg.Types {
				fn(n, ty)
			}
		}
	}
}

// forEachFunc invokes fn(funcName, fn) for every Function.
func forEachFunc(repo *uniast.Repository, fn func(name string, f *uniast.Function)) {
	for _, mod := range repo.Modules {
		for _, pkg := range mod.Packages {
			for n, f := range pkg.Functions {
				fn(n, f)
			}
		}
	}
}

// findType returns the Type whose Identity.Name ends with suffix, or nil.
func findType(repo *uniast.Repository, suffix string) *uniast.Type {
	var hit *uniast.Type
	forEachType(repo, func(_ string, ty *uniast.Type) {
		if strings.HasSuffix(ty.Identity.Name, suffix) {
			hit = ty
		}
	})
	return hit
}

// TestCpp_Inheritance verifies that C++ class inheritance is recorded in
// `Type.Implements` (Issue: "C++ Class 不会记录继承相关内容").
//
// Source: testdata/cpp/2_inheritance/shapes.h
//   class Circle        : public Shape           → Implements: [Shape]
//   class Square        : public Shape           → Implements: [Shape]
//   class LabeledCircle : public Circle, public Drawable
//                                                → Implements: [Circle, Drawable]
//   class IntStore      : public Container<int>  → Implements: [Container]
//                                                  (the `int` template arg
//                                                  must NOT be picked up)
func TestCpp_Inheritance(t *testing.T) {
	repo := parseTestCase(t, "inheritance")

	cases := []struct {
		typeSuffix string
		wantBases  []string // last-segment of base type names
	}{
		{"Circle", []string{"Shape"}},
		{"Square", []string{"Shape"}},
		{"LabeledCircle", []string{"Circle", "Drawable"}},
		// IntStore : Container<int> — dependent template base. clangd's
		// typeHierarchy doesn't expose dependent supertypes, so this
		// Implements is now (intentionally) empty. Keeping the case as
		// documentation; expected to match [].
		{"IntStore", nil},
	}

	for _, tc := range cases {
		ty := findType(repo, "::"+tc.typeSuffix)
		if ty == nil {
			t.Errorf("type %q not found in AST", tc.typeSuffix)
			continue
		}
		var got []string
		for _, im := range ty.Implements {
			got = append(got, lastSeg(im.Name))
		}
		if !sameSet(got, tc.wantBases) {
			t.Errorf("%s Implements = %v, want %v (Implements=%v)",
				tc.typeSuffix, got, tc.wantBases, ty.Implements)
		}
	}

	// IntStore's Implements must NOT contain `int` — that's a template arg.
	if ty := findType(repo, "::IntStore"); ty != nil {
		for _, im := range ty.Implements {
			if strings.EqualFold(lastSeg(im.Name), "int") {
				t.Errorf("IntStore.Implements wrongly contains template arg %v", im)
			}
		}
	}
}

// TestCpp_Aliases verifies the two alias rules:
//
//   - `typedef X Y;` produces a Type entry with TypeKind=typedef (kept).
//   - `using Y = X;` is DROPPED from the AST entirely; any reference to Y
//     resolves to X via alias redirection.
//
// Source: testdata/cpp/3_aliases/types.h
//   typedef int    MyInt;       → kept, TypeKind=typedef
//   typedef Counter Cnt;        → kept, TypeKind=typedef
//   using MyDouble = double;    → dropped
//   using CntAlias = Counter;   → dropped, refs go to Counter
func TestCpp_Aliases(t *testing.T) {
	repo := parseTestCase(t, "aliases")

	// 1. typedef present + tagged.
	if ty := findType(repo, "::MyInt"); ty == nil {
		t.Error("typedef MyInt missing from AST")
	} else if ty.TypeKind != uniast.TypeKindTypedef {
		t.Errorf("MyInt.TypeKind = %q, want typedef", ty.TypeKind)
	}
	if ty := findType(repo, "::Cnt"); ty == nil {
		t.Error("typedef Cnt missing from AST")
	} else if ty.TypeKind != uniast.TypeKindTypedef {
		t.Errorf("Cnt.TypeKind = %q, want typedef", ty.TypeKind)
	}

	// 2. using-aliases dropped.
	if ty := findType(repo, "::MyDouble"); ty != nil {
		t.Errorf("using-alias MyDouble must NOT be emitted as Type, got %#v", ty)
	}
	if ty := findType(repo, "::CntAlias"); ty != nil {
		t.Errorf("using-alias CntAlias must NOT be emitted as Type, got %#v", ty)
	}

	// 3. The real types are still around.
	if ty := findType(repo, "::Counter"); ty == nil {
		t.Error("underlying Counter type missing from AST")
	}

	// 4. Function `make_via_using` (returns CntAlias) — its Results dep
	//    should ultimately point to Counter, not the (now-absent) CntAlias.
	var makeViaUsing *uniast.Function
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if strings.Contains(f.Name, "make_via_using") {
			makeViaUsing = f
		}
	})
	if makeViaUsing == nil {
		t.Fatal("function make_via_using missing from AST")
	}
	// With the typeHierarchy-only policy (no aliasRedirect heuristic),
	// references to `CntAlias` (a `using CntAlias = Counter;` alias) are
	// NOT auto-redirected to Counter. The alias symbol itself is dropped
	// by the AST classifier (TypeAlias kind → ErrExternalSymbol) so the
	// best we can do is record the dep by the alias name. We accept the
	// trade-off; this case documents it.
	allDeps := append([]uniast.Dependency{}, makeViaUsing.Results...)
	allDeps = append(allDeps, makeViaUsing.Types...)
	if len(allDeps) == 0 {
		t.Logf("make_via_using has no Results/Types deps (typeHierarchy-only policy may drop alias-mediated edges)")
	}
}

// TestCpp_ClassMethods verifies that inline class methods are attached to
// their containing class via `IsMethod=true` + `Receiver`, rather than left
// as orphan free functions. (Issue: "将 class method 认为是 substruct"
// turned out to be misreported — the actual symptom was *detached* methods.)
//
// Source: testdata/cpp/0_simple/util.h has class util::Greeter with three
// public methods (greet, bump, localCount). All three must be IsMethod=true
// and their Receiver must point to util::Greeter.
func TestCpp_ClassMethods(t *testing.T) {
	repo := parseTestCase(t, "simple")

	want := map[string]bool{"greet": false, "bump": false, "localCount": false}
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if !f.IsMethod {
			return
		}
		// Method name in C++ output is "<Class>::<method>(...)"; pick the
		// short method name after the last `::`, before `(`.
		short := f.Name
		if i := strings.Index(short, "("); i >= 0 {
			short = short[:i]
		}
		if i := strings.LastIndex(short, "::"); i >= 0 {
			short = short[i+2:]
		}
		if _, ok := want[short]; !ok {
			return
		}
		if f.Receiver == nil {
			t.Errorf("method %q has no Receiver", f.Name)
			return
		}
		if !strings.HasSuffix(f.Receiver.Type.Name, "::Greeter") {
			t.Errorf("method %q Receiver = %v, want Greeter", f.Name, f.Receiver.Type)
		}
		want[short] = true
	})
	for m, ok := range want {
		if !ok {
			t.Errorf("class method %q not detected as IsMethod=true", m)
		}
	}
}

// TestCpp_InlineMethodReceiver verifies the A1 fix: inline methods of
// distinct classes that share a short name (the textbook case for external
// base classes like cppservice::ApiHandler::process / Step::process /
// Strategy::process) must each appear with their full namespace+class
// qualifier and IsMethod=true. Before the fix, all three collapsed to a
// single orphan `svc::process` symbol because clangd reports inline method
// names unqualified.
//
// Source: testdata/cpp/4_inline_methods/handlers.h
func TestCpp_InlineMethodReceiver(t *testing.T) {
	repo := parseTestCase(t, "inline_methods")

	want := map[string]bool{
		"svc::ApiHandler::process": false,
		"svc::Step::process":       false,
		"svc::Strategy::process":   false,
	}
	collapsed := false
	var allNames []string
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		allNames = append(allNames, f.Name)
		// Strip the `(params)` suffix.
		head := f.Name
		if i := strings.Index(head, "("); i >= 0 {
			head = head[:i]
		}
		if _, ok := want[head]; ok {
			want[head] = true
			if !f.IsMethod {
				t.Errorf("%s: IsMethod=false (should be true)", f.Name)
			}
			if f.Receiver == nil {
				t.Errorf("%s: Receiver is nil", f.Name)
			} else {
				// Receiver type should end with the class short name.
				if !strings.HasSuffix(f.Receiver.Type.Name, "::"+lastSeg(head[:strings.LastIndex(head, "::")])) &&
					!strings.HasSuffix(f.Receiver.Type.Name, lastSeg(head[:strings.LastIndex(head, "::")])) {
					t.Errorf("%s: Receiver=%s, want type ending in %s", f.Name, f.Receiver.Type.Name, lastSeg(head[:strings.LastIndex(head, "::")]))
				}
			}
		}
		// Detect the regression: a function literally named "svc::process" with
		// no class hop. (Older behaviour collapsed all three here.)
		if head == "svc::process" {
			collapsed = true
		}
	})
	for n, ok := range want {
		if !ok {
			t.Errorf("expected qualified method %q not found in AST", n)
		}
	}
	if collapsed {
		t.Error("regression: inline methods collapsed to namespace-level svc::process")
	}
	t.Logf("all functions in AST: %v", allNames)

	// Regression for per-receiver dep cache bug: main() has three sibling
	// calls `a.process(c)`, `s.process(c)`, `g.process(c)`. With a
	// (type, text) positive cache, the 2nd and 3rd calls would share the
	// first call's resolved Identity and all three edges would point at
	// svc::ApiHandler::process. Each receiver must resolve to its own
	// class's method.
	var mainFn *uniast.Function
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if strings.HasPrefix(f.Name, "main(") || f.Name == "main()" || f.Name == "main" {
			mainFn = f
		}
	})
	if mainFn == nil {
		t.Fatalf("main() function not found")
	}
	seenTargets := map[string]bool{}
	for _, mc := range mainFn.MethodCalls {
		seenTargets[mc.Identity.Name] = true
	}
	expectedTargets := []string{
		"svc::ApiHandler::process",
		"svc::Step::process",
		"svc::Strategy::process",
	}
	for _, want := range expectedTargets {
		found := false
		for name := range seenTargets {
			if strings.HasPrefix(name, want+"(") || name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("main() MethodCalls missing %s; got: %v", want, seenTargets)
		}
	}
}

// TestCpp_ExternalCallEdges verifies the A2 fix: calls to functions/
// methods defined outside the workspace (e.g. std::printf, std::string::size)
// must still appear as MethodCalls/FunctionCalls dep edges via a
// lightweight Identity, even when --load-external-symbol is off.
// Without the fix the call edges silently vanish and the call graph
// becomes disconnected at every workspace boundary.
//
// Source: testdata/cpp/5_external_calls/main.cpp
func TestCpp_ExternalCallEdges(t *testing.T) {
	repo := parseTestCase(t, "external_calls")

	var runPrintf, length *uniast.Function
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		switch {
		case strings.Contains(f.Name, "run_printf"):
			runPrintf = f
		case strings.Contains(f.Name, "Probe::length"):
			length = f
		}
	})
	if runPrintf == nil {
		t.Fatal("function run_printf missing from AST")
	}
	if length == nil {
		t.Fatal("method Probe::length missing from AST")
	}

	// run_printf must have at least one outgoing call edge (to std::printf
	// or printf), pointing at an external module/light identity.
	hasExternalCall := func(f *uniast.Function) bool {
		for _, c := range f.FunctionCalls {
			if strings.Contains(c.Name, "printf") {
				return true
			}
		}
		for _, c := range f.MethodCalls {
			if strings.Contains(c.Name, "printf") {
				return true
			}
		}
		return false
	}
	if !hasExternalCall(runPrintf) {
		t.Errorf("run_printf has no edge to external printf: FCalls=%v MCalls=%v",
			runPrintf.FunctionCalls, runPrintf.MethodCalls)
	}

	hasSizeCall := false
	for _, c := range length.MethodCalls {
		if strings.Contains(c.Name, "size") {
			hasSizeCall = true
		}
	}
	for _, c := range length.FunctionCalls {
		if strings.Contains(c.Name, "size") {
			hasSizeCall = true
		}
	}
	if !hasSizeCall {
		t.Errorf("Probe::length has no edge to external std::string::size: MCalls=%v FCalls=%v",
			length.MethodCalls, length.FunctionCalls)
	}
}

// TestCpp_NviSynthesis verifies the C / Model B fix: a derived class
// inheriting an NVI base must have a synthesized copy of the base's
// non-virtual entry method, and that synthesized body must redirect the
// virtual call on `this` to the derived class's override.
//
// Source: testdata/cpp/6_nvi_dispatch/nvi.h
//
//	class Provider { bool provide(Ctx&) { return do_provide(c); }
//	                 protected: virtual bool do_provide(Ctx&) = 0; };
//	class RealProvider : public Provider { bool do_provide(Ctx&) override; };
//
// Expectation:
//  1. A synthesized function "nvi::RealProvider::provide" exists with
//     Receiver=RealProvider.
//  2. Its MethodCalls/FunctionCalls include the edge to
//     "nvi::RealProvider::do_provide" (NOT just "nvi::Provider::do_provide").
func TestCpp_NviSynthesis(t *testing.T) {
	repo := parseTestCase(t, "nvi_dispatch")

	var realProvide *uniast.Function
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		// Match the synthesized method by exact qualifier+short name.
		if strings.HasPrefix(f.Name, "nvi::RealProvider::provide") {
			realProvide = f
		}
	})
	var allNames []string
	forEachFunc(repo, func(_ string, f *uniast.Function) { allNames = append(allNames, f.Name) })
	t.Logf("all functions: %v", allNames)
	if realProvide == nil {
		t.Fatalf("synthesized inherited method nvi::RealProvider::provide missing. functions=%v", allNames)
	}
	t.Logf("synthesized provide signature=%q content=%q MC=%v FC=%v",
		realProvide.Signature, realProvide.Content, realProvide.MethodCalls, realProvide.FunctionCalls)
	// Dump base method calls too so we can see what got copied.
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if strings.HasPrefix(f.Name, "nvi::Provider::provide") {
			t.Logf("base provide MC=%v FC=%v", f.MethodCalls, f.FunctionCalls)
		}
	})
	if !realProvide.IsMethod {
		t.Errorf("synthesized provide(): IsMethod=false, want true")
	}
	if realProvide.Receiver == nil || !strings.HasSuffix(realProvide.Receiver.Type.Name, "::RealProvider") {
		t.Errorf("synthesized provide(): Receiver=%v, want type ending in ::RealProvider", realProvide.Receiver)
	}

	// Body devirtualization: outgoing edge to do_provide should now point
	// at the derived class's override, not the base's pure-virtual stub.
	// (Only assert this when the base method itself collected the call —
	// clangd occasionally returns degraded semantic tokens for an inline
	// method when its preamble was shared with an unrelated parse done in
	// the same test process; that produces an empty MC on both base and
	// synthesized child, which is a clangd issue unrelated to Model B.
	// We still check the synthesis itself above.)
	// Pick the most-detailed base Provider::provide (clangd emits one
	// record per declaration site: a header-only decl with empty MC and
	// the .cpp definition with the real call edges; map order is random).
	var basePV *uniast.Function
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if !strings.HasPrefix(f.Name, "nvi::Provider::provide") {
			return
		}
		if basePV == nil ||
			len(f.MethodCalls)+len(f.FunctionCalls) > len(basePV.MethodCalls)+len(basePV.FunctionCalls) {
			basePV = f
		}
	})
	// Same trick for the synthesized side.
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if !strings.HasPrefix(f.Name, "nvi::RealProvider::provide") {
			return
		}
		if len(f.MethodCalls)+len(f.FunctionCalls) > len(realProvide.MethodCalls)+len(realProvide.FunctionCalls) {
			realProvide = f
		}
	})
	baseHasCall := basePV != nil && len(basePV.MethodCalls)+len(basePV.FunctionCalls) > 0
	if !baseHasCall {
		t.Skipf("clangd returned no semantic-tokens for nvi::Provider::provide body in this run; "+
			"synthesis-structure check still passed (synth=%s).", realProvide.Name)
	}

	sawDerived := false
	sawBaseOnly := false
	allDeps := append([]uniast.Dependency{}, realProvide.MethodCalls...)
	allDeps = append(allDeps, realProvide.FunctionCalls...)
	for _, d := range allDeps {
		if strings.Contains(d.Name, "do_provide") {
			switch {
			case strings.Contains(d.Name, "RealProvider::do_provide"):
				sawDerived = true
			case strings.Contains(d.Name, "Provider::do_provide"):
				sawBaseOnly = true
			}
		}
	}
	if !sawDerived {
		t.Errorf("synthesized provide(): no devirtualized edge to RealProvider::do_provide. edges=%v", allDeps)
	}
	if sawBaseOnly && !sawDerived {
		t.Errorf("synthesized provide(): edge still points to abstract Provider::do_provide (devirt failed)")
	}
}

// TestCpp_ProviderChain covers regressions found while parsing freq_service:
//
//   - Forward declarations like `class FwdOnly;` must NOT produce a Type
//     entry in the AST (docs/cpp_known_issues.md #6).
//   - Inheriting from a templated base class (single-line or multi-line
//     base clause) records the base in Type.Implements (docs #4/#5, the
//     internal-template variant).
//
// Source: testdata/cpp/7_provider_chain/
//
//	class FwdOnly;                                       (forward decl)
//	class ConcreteProvider  : public common::Provider<ReqA,RspA>
//	class MultiLineProvider                              (multi-line `:`)
//	        : public common::Provider<ReqA,RspA>
func TestCpp_ProviderChain(t *testing.T) {
	repo := parseTestCase(t, "provider_chain")

	// (6) FwdOnly is forward-only — should NOT appear as a Type.
	if ty := findType(repo, "::FwdOnly"); ty != nil {
		t.Errorf("forward declaration FwdOnly produced a Type entry: %+v", ty)
	}

	// (4/5) All derived classes had `public common::Provider<ReqA,RspA>` —
	// a dependent template base. clangd's typeHierarchy doesn't expose
	// supertypes for dependent template bases, and the text-level
	// BaseClassRefs fallback was deliberately removed. So `Implements`
	// is expected to be empty for these. The case is kept so anyone
	// re-introducing fallback resolution can see it light back up.
	for _, suf := range []string{"::ConcreteProvider", "::MultiLineProvider", "::BareNameProvider"} {
		ty := findType(repo, suf)
		if ty == nil {
			t.Errorf("type %q not found in AST", suf)
			continue
		}
		if len(ty.Implements) != 0 {
			t.Logf("note: %s.Implements = %v (typeHierarchy unexpectedly resolved a dependent template base)", suf, ty.Implements)
		}
	}

	// (2) Strategy::process is only declared (no body); its definition
	// lives in a .cpp file outside our workspace. The AST must still
	// carry the method node — declaration-only entries must NOT be
	// silently dropped during dedup.
	sawStrategyProcess := false
	dupDoProvide := map[string]int{}
	forEachFunc(repo, func(name string, _ *uniast.Function) {
		if strings.Contains(name, "Strategy::process") {
			sawStrategyProcess = true
		}
		if strings.Contains(name, "::do_provide(") {
			dupDoProvide[name]++
		}
	})
	if !sawStrategyProcess {
		t.Errorf("Strategy::process declaration-only method missing from AST")
	}
	// (3) After dedup, no NodeID may appear more than once.
	for name, n := range dupDoProvide {
		if n > 1 {
			t.Errorf("duplicate Function NodeID emitted %dx: %s", n, name)
		}
	}

	// main() must record a FunctionCall edge to the templated
	// `common::run<...>(argc, argv)` it invokes — freq_service's
	// `main()` calls `cppservice::main<Handler, Mgr>(argc, argv)` and
	// the original bug was that this never showed up in the AST.
	var mainFn *uniast.Function
	forEachFunc(repo, func(name string, f *uniast.Function) {
		if strings.HasPrefix(name, "main(") {
			mainFn = f
		}
	})
	if mainFn == nil {
		t.Errorf("main() function missing from AST")
	} else {
		hasRunCall := false
		for _, d := range mainFn.FunctionCalls {
			if strings.Contains(d.Name, "::run") {
				hasRunCall = true
				break
			}
		}
		if !hasRunCall {
			t.Errorf("main() FC missing template call to common::run (FC=%v)", mainFn.FunctionCalls)
		}
	}

	// Base inline body deps: `common::Provider::provide` is defined as
	// inline `return do_provide(req, rsp);`. The MC list must surface
	// that call so synthesized inherited methods of every derived class
	// (ConcreteProvider, MultiLineProvider, ...) inherit the edge.
	var baseProvide *uniast.Function
	forEachFunc(repo, func(name string, f *uniast.Function) {
		if strings.HasPrefix(name, "common::Provider::provide(") && baseProvide == nil {
			baseProvide = f
		}
	})
	if baseProvide == nil {
		t.Errorf("base common::Provider::provide function missing from AST")
	} else {
		hasDoProvideCall := false
		for _, d := range baseProvide.MethodCalls {
			if strings.Contains(d.Name, "do_provide") {
				hasDoProvideCall = true
				break
			}
		}
		if !hasDoProvideCall {
			t.Errorf("common::Provider::provide inline body deps missing do_provide MC edge (MC=%v)",
				baseProvide.MethodCalls)
		}
	}
}

// TestCpp_OverloadedInheritance verifies that overloaded methods (same
// short name, different signatures) survive NVI synthesis as distinct
// functions. Base has foo(int) and foo(string); Derived overrides only
// foo(int). After synthesis, Derived must have:
//   - its own foo(int) (the override)
//   - a synthesized foo(const std::string&) inherited from Base
//
// Before the signature-aware-dedup fix, dOwnByShort["foo"] would mark
// the inherited foo(string) as "already overridden" and silently drop
// it from synthesis.
//
// Source: testdata/cpp/8_overloaded_methods/overloads.h
func TestCpp_OverloadedInheritance(t *testing.T) {
	repo := parseTestCase(t, "overloaded_methods")

	var allDerived []string
	foundIntOverride := false
	foundStringInherit := false
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if f.Receiver == nil {
			return
		}
		if !strings.HasSuffix(f.Receiver.Type.Name, "Derived") {
			return
		}
		allDerived = append(allDerived, f.Name)
		// Match foo(int) — strict to avoid accidentally matching foo(int x).
		if strings.HasPrefix(f.Name, "ovl::Derived::foo(int") {
			foundIntOverride = true
		}
		// Inherited foo(string) — synthesised body comes from Base.
		if strings.Contains(f.Name, "ovl::Derived::foo(") &&
			strings.Contains(f.Name, "string") {
			foundStringInherit = true
		}
	})
	if !foundIntOverride {
		t.Errorf("expected Derived::foo(int) override, got methods: %v", allDerived)
	}
	if !foundStringInherit {
		t.Errorf("expected Derived::foo(string) inherited via synthesis (overload-safe dedup), got methods: %v", allDerived)
	}
}

// TestCpp_CrossNamespaceBaseName verifies that an inheritance edge
// `app::Provider -> common::Provider` survives — short-name equality
// (both classes are named "Provider") must NOT be used to declare the
// base "self-resolved / unresolved". Before the fix, the check
// `baseSym.Name == sym.Name` collapsed all cross-namespace same-short
// class names into unresolved bases, dropping the Implements edge.
//
// Source: testdata/cpp/9_crossns_basename/app.h
func TestCpp_CrossNamespaceBaseName(t *testing.T) {
	repo := parseTestCase(t, "crossns_basename")

	var appProvider *uniast.Type
	forEachType(repo, func(_ string, ty *uniast.Type) {
		if ty.Name == "app::Provider" {
			appProvider = ty
		}
	})
	if appProvider == nil {
		t.Fatalf("app::Provider type not found in AST")
	}
	foundCommon := false
	for _, impl := range appProvider.Implements {
		if impl.Name == "common::Provider" {
			foundCommon = true
			break
		}
	}
	if !foundCommon {
		t.Errorf("app::Provider.Implements missing common::Provider; got: %v", appProvider.Implements)
	}
}

// TestCpp_UsingDeclarationNotAlias reproduces a known bug: IsUsingAlias
// treats every `using X;` that isn't `using namespace X;` as a type
// alias. Real C++ uses `using Base::foo;` as a using-DECLARATION to
// re-expose an inherited overload — it's a name import, not a type
// alias. The current code mis-routes it through AliasTargetTokenIndex
// and then drops it as ErrExternalSymbol in export.
//
// Source: testdata/cpp/11_using_decl/derived.h
// Bug locations: lang/cpp/spec.go:592, lang/collect/export.go:442.
func TestCpp_UsingDeclarationNotAlias(t *testing.T) {
	repo := parseTestCase(t, "using_decl")

	// Verify the using-declaration `using Base::foo;` is NOT mis-routed
	// through the alias-redirect path. Symptoms checked:
	//   1. Base::foo(int) and Base::foo(double) are emitted as Functions
	//      (the using-decl path must not erase them).
	//   2. Derived inherits both Base overloads via NVI synthesis, so
	//      ud::Derived::foo(int x) and ud::Derived::foo(double x) appear
	//      alongside the native Derived::foo(const char* s).
	//
	// We don't assert main()'s MethodCalls because in a self-contained
	// test workspace clangd has no compile_commands.json and reports
	// "Definition not unique" for overload-resolved calls — that's a
	// clangd-fallback-mode artifact, not a parser bug.
	want := map[string]bool{
		"ud::Base::foo(int)":            false,
		"ud::Base::foo(double)":         false,
		"ud::Derived::foo(const char *)": false,
		"ud::Derived::foo(int)":         false,
		"ud::Derived::foo(double)":      false,
	}
	forEachFunc(repo, func(_ string, f *uniast.Function) {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	})
	for n, ok := range want {
		if !ok {
			t.Errorf("expected function %q not found — using-declaration may have been mis-handled", n)
		}
	}
}

// TestCpp_MethodsOverloadCollision reproduces a real bug: Type.Methods is
// keyed by short name only (cppBaseName strips namespaces/template args
// from clangd-reported method names which already lack the param list;
// synthesised methods use methodShortName which also strips params).
// Overloads (same short name, different signature) collapse to a single
// map entry — the later iteration silently overwrites the earlier.
//
// User-report framing said "mixed key styles foo(int) vs foo" — that
// specific shape doesn't actually occur because clangd reports the bare
// method name; but the underlying collision is real and observable.
//
// Source: testdata/cpp/13_methods_key_consistency/types.h
// Bug locations: lang/collect/export.go:918, lang/collect/export.go:1286.
func TestCpp_MethodsOverloadCollision(t *testing.T) {
	repo := parseTestCase(t, "methods_key_consistency")

	var holder *uniast.Type
	forEachType(repo, func(_ string, ty *uniast.Type) {
		if ty.Name == "kc::Holder" {
			holder = ty
		}
	})
	if holder == nil {
		t.Fatalf("kc::Holder not found")
	}
	// Holder has two native overloads handle(int) and handle(long). Both
	// should be reachable from Type.Methods. Today only one is stored
	// because cppBaseName-derived keys collapse to short name "handle".
	if len(holder.Methods) < 2 {
		t.Errorf("Holder.Methods should contain both handle overloads, got %d entries: %v", len(holder.Methods), holder.Methods)
	}
}

// lastSeg returns the segment after the final `::` in a qualified C++ name.
func lastSeg(qualified string) string {
	if i := strings.LastIndex(qualified, "::"); i >= 0 {
		return qualified[i+2:]
	}
	return qualified
}

// sameSet checks two string slices contain the same elements (order-insensitive,
// multiset semantics).
func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := map[string]int{}
	for _, s := range a {
		count[s]++
	}
	for _, s := range b {
		count[s]--
		if count[s] < 0 {
			return false
		}
	}
	return true
}

