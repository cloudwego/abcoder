# C++ Collector Known Issues

记录 freq_service v7 parse 之后用户发现的 6 个问题。每条带：现象、复现位置、根因猜想、修复进度。

## 1. main.cpp 没有 method（实为：模板函数调用没进 FC）

**现象（修订）**：`main` 函数节点是存在的，但 FunctionCalls / MethodCalls 都是 0。content 里能看到 `cppservice::main<...>(argc, argv)` 模板调用。

**实证**：v7 AST `service/main.cpp` 唯一函数 `main(int argc, char** argv)`，FC=0, MC=0。Content `return cppservice::main<idl::data::freq::archon::FreqServer, ...>(argc, argv);`。

**根因猜想**：模板函数调用的 token 在 clangd 里类型是 `function`/`template`，未被识别为 reference token，导致 Definition 查询不发生，dep 收集失败。

## 2. `cppservice::Strategy::process(RequestContext& ctx)` 没有定义

**现象**：节点存在，但其 Content 字段为空 / FileLine 指向不存在 / 是 declaration-only 残留。

**复现**：v7 AST 搜 `Strategy::process` 节点。Strategy 是 freq_service 内部基类（参考 NVI 模式）。

**根因猜想**：Strategy 是一个非纯虚基类，header 有声明，cpp 中有定义；但 abcoder 把 cpp 里的定义弄丢了，剩了 header 那份；或 header 那份被 declaration-only 过滤吃掉，导致整条记录消失。

## 3. `FeathubProvider::cppservice::Provider::provide(...)` 既无 call 也无 construct

**现象**：节点存在，但 FunctionCalls、MethodCalls、Types（构造调用） 全空。

**复现**：v7 AST 搜 `FeathubProvider::cppservice::Provider::provide`。

**根因猜想**：(a) 名字形态怪异（`Class::OtherNS::OtherClass::method`），说明 NodeID 拼装出错；(b) 节点是 NVI 合成的 ImplHead 包裹件，原 deps 没有挂上来；(c) ImplHead 路径绕过了 deps 收集。

## 4. Provider 相关的没有 Implement

**现象**：`XxxProvider extends Provider` 时，AST 上 XxxProvider Type 的 Implements 字段为空，断了继承链。

**复现**：v7 AST 看 freq_service Provider 派生类。

**根因猜想**：`Provider` 基类是模板（`template<class Req, class Rsp> class Provider`），BaseClassTokens 可能在模板参数里走偏；或基类 token 被 isReferenceTypeToken 判错；或 Implements 写入 races。

## 5. 继承时 `Provider::provide` 应改名为 `XXXProvider::provide`，很多没改

**现象**：派生类有自己的 override，但在 AST 中仍以基类 NodeID（`Provider::provide`）出现，没绑到派生类 NodeID 上。

**复现**：v7 AST 搜 `Provider::provide` 节点。应该看到 `XxxProvider::provide` 而非泛型基类。

**根因猜想**：CppSpec 在 method 归属时把 method 挂到了 SKMethod 报告的 containerName，而 clangd 偶尔会把 override 报成 base class 的 method；也可能是 Method.Receiver 解析失败 fallback 到了 base。

**实证补充（问题 4/5 共同的）**：在 v7 freq_service 中检查所有 Provider 派生类的 Implements 字段，只有继承裸 `Provider`（非模板）的 FeathubProvider 的 Implements 是 `['cppservice::Provider']`，其余继承 `SimpleProvider<X,Y>`、`CircuitBreakerProvider<X,Y>` 的派生类 Implements 全部为空 — 说明 BaseClassTokens 在模板基类场景下没找到正确的 base type token（很可能 isReferenceTypeToken 把模板参数当成 base type，或在 `<...>` 区间外的 base 名字漏过）。

## 6. 前向声明出现在 Types 里

**现象**：`class AwemeUserProfileFeatureProvider;` 这类前向声明产生了一个 Type 节点（无字段、无方法），跟真正的 class 定义并列。

**复现**：v7 AST 搜该名字，会看到一个 Type 但 Methods/SubStruct/Fields 全空。

**根因猜想**：clangd 会把 forward decl 报成单独的 DocumentSymbol（SKClass，Range 只覆盖 `class X;` 这一行）。export.go 没有区分 forward-only 与定义，两者都写入了 Types 表。

**实证**：单单 `service/core/freq_resource_manager.h` 里就有 107 个 Type 节点，**106 个**是前向声明（Methods=0、Fields=0、Implements=[]、content 形如 `class XxxProvider`，注意末尾 `;` 被 declarationText 截掉了）。仅 1 个是真定义。

---

## 修复计划

1. 调查阶段：用 v7 AST 把每条问题的具体节点 dump 出来，看 Content / Identity / 字段。
2. 修复阶段：分条改 collect / spec / export，并优先动作小的（forward decl 过滤、main 函数过滤误伤、declaration-only 误伤 main/Strategy）；继承重命名/Implements/ImplHead-deps 涉及 NVI 合成与 receiver 解析，要小心。
3. 测试阶段：每修一条，在 `testdata/cpp/` 下增加 minimal repro 样本，在 `lang/parse_cpp_test.go` 加 assert，回归测试中跑通。

---

## 修复进度

### ✓ 已修

- **问题 6（forward decl 误入 Types）** — `lang/collect/export.go` 在 SKClass/SKStruct emit 之前判断 content 是否以 `class `/`struct ` 开头且不含 `{`，是则跳过。typedef 不受影响。
- **问题 4/5（模板基类 Implements）** — clangd 对外部未解析的模板基类不上报 base token，旧的 token-index 路径漏掉它们。`lang/cpp/spec.go` 新增 `BaseClassRefs(sym)` 从源码 `declarationText` 直接 parse base 名字 + 计算文件位置，`lang/collect/collect.go::collectImpl` 优先用该路径；当 Definition 解不出（fallback 回 self）时，记录到 `cppUnresolvedBases`；`lang/collect/export.go` 把它们转成 light Identity 加到 Type.Implements。
- **问题 2（Strategy::process declaration-only 节点缺失）** — 之前的 declaration-only 过滤把外部头里只有 declaration 的 method 一并丢弃。改用 cross-pkg dedup：保留 *body 那份*；当只有 declaration 没有 definition 时（外部 cpp 未加载、纯虚等），保留 declaration。Collector 新增 `cppFnEmittedBody`/`cppFnLocation` 两个 map 来识别原始 hasBody 状态（不受 ImplHead 包装影响）。
- **NodeID 拼装 bug（问题 3 的一部分）** — `synthesizeInheritedMethodsCpp` 用 `LastIndex("::")` 找 receiver 分割点，但 `(` 之后的参数（`const std::string&`）里也有 `::`，落到错误位置。改成只在 `(` 之前的 head 内找。

- **问题 3（NVI inline body 内 deps 缺失）** ✓ 修复
  - 根因：外部 `LoadExternalSymbol` 加载的 method/function sym 进入 `c.syms`，但 `collectDepsForEntity` 只对 workspace 内 `entity_syms` 调用。`needProcessExternal` 对 C++ 又要求 `SKObject`（clangd 永远不报），所以外部 method 永远无 deps。
  - 修复：`Collector` 增加 `cppExtDepsQueue`，在 `getSymbolByLocation` 的 external-load 分支里把每个 SKMethod/SKFunction sym 入队；`Collect` 结尾跑一遍批量 `collectDepsForEntity`（并发，去重）。直接 inline 会触发 `getSymbolByToken` → 新外部 load → 新 `collectDepsForEntity` 的无界递归。
  - v17 验证：`cppservice::Provider::provide` MC=1 → `cppservice::Remote::call`；合成的 `cppservice::freq::FeathubProvider::provide` 继承到同一边。

- **header-inline 方法 body deps 不收集** ✓ 修复
  - 根因：method 的 `functionInfo.Method.Receiver.Location` fallback 时被设成 *整个 receiver class body* range（`collect.go:2462` 的 GetParent 路径）。`collectDepsForEntity` 用 `Receiver.Location.Include(token.Location)` skip receiver-qualifier token —— 对 inline-in-class method，class body 包含整个 method body，body 内 *所有* token 都被这条规则误 skip。
  - 修复：`collect.go::collectDepsForEntity` 区分两种情况：当 sym.Location 落在 receiver class body 内（即 inline），只 skip 字面匹配 receiver class 名字的 token；当 sym 在 receiver class body 外（即 out-of-line），保留原 Include-based skip 行为。

- **空 package 清理** ✓ — 修复 inline body deps 之后，dedup 把 .cpp 里的 method 全部移到 .h pkg，留下 *空的 .cpp pkg*。export 末尾扫一遍 `repo.Modules`，删 `len(Functions)==len(Types)==len(Vars)==0` 的 pkg。
- **Type.Methods 回填合成 method** ✓ — `synthesizeInheritedMethodsCpp` 末尾把合成的 NodeID 加到 derived type 的 Methods map（之前只写到 pkg.Functions，Type 的 Methods 字段缺这条 entry）。
- **method 归属重新指到 .h pkg** ✓ — 当存在同 NodeID 的 .h decl + .cpp def 时，合并：保留 .h 的 pkg 位置，但把 content/FC/MC 等 body 字段从 .cpp 覆盖过来。

### 后续修复

- **问题 1（main 模板函数调用未进 FC）** ✓ 修复
  - 真正根因不在 `IsEntityToken`（`run` token type=`function` 是 entity OK），而是 inline body deps 的 receiver-skip 副作用 — `main` 本身没受影响，但同套修复让 inline body 内 token 不再被误 skip 后，所有 function-call edge（含模板）都能正确收集。testdata 7 main() 调 `common::run<X,Y>(argc, argv)` 现在产生 FC=1 → `common::run`。

- **`using NS::Name;` 别名 NodeID 形态错乱** ✓ 修复
  - 根因：`using common::Provider;` 这种 using-declaration 被 clangd 当成一个 phantom `Class` DocumentSymbol，Range 只覆盖 `Provider` 8 字符；BaseClassRefs 的 fakeTok Definition 命中这个 phantom sym，lightIdentityForExternal 拼出 `app::Provider::Provider`。
  - 修复 1：`spec.go::IsUsingAlias` 扩展支持 `using NS::Name;`（不要求有 `=`）。
  - 修复 2：`collect.go::collectImpl` 中拿到 baseSym 后检测 *单行 + 字符数 ≈ name 长* 的 narrow Range —— 是 alias decl 标志 —— 再调一次 `c.cli.Definition` follow 到真目标，最多 4 跳。
  - 修复 3：新增 `Collector.resolveAlias`（cyclic-safe）走 `c.aliasRedirect` 链。
  - testdata 7 `BareNameProvider` 现在 `Implements=['common::Provider']`、`Methods=['do_provide', 'provide']`。
