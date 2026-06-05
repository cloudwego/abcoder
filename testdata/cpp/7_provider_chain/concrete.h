#ifndef CONCRETE_H
#define CONCRETE_H

#include "provider.h"

namespace app {

// Forward declaration — must NOT produce a Type entry in the AST.
class FwdOnly;

struct ReqA {
    int v;
};
struct RspA {
    int v;
};

// Templated base class with concrete template args — Implements should
// contain `Provider`, not the template args ReqA/RspA.
class ConcreteProvider : public common::Provider<ReqA, RspA> {
public:
    ConcreteProvider() = default;

protected:
    bool do_provide(ReqA& req, RspA& rsp) override;
};

// Multi-line base clause mimicking the freq_service style — the `:` lives
// on its own line. Implements must still resolve to `Provider`.
class MultiLineProvider
        : public common::Provider<ReqA, RspA> {
public:
    MultiLineProvider() = default;

protected:
    bool do_provide(ReqA& req, RspA& rsp) override;
};

} // namespace app

// Sibling namespace import via `using` — mirrors how freq_service refers to
// `CircuitBreakerProvider` without an explicit `cppservice::` prefix.
namespace app {
using common::Provider;

namespace inner {
struct ReqB { int v; };
struct RspB { int v; };

// This is the freq_service-style: base name appears WITHOUT namespace
// prefix (resolved via `using`) and template args carry their own
// namespaces. Implements should still resolve to `Provider`.
class BareNameProvider
        : public Provider<ReqB, RspB> {
public:
    BareNameProvider() = default;

protected:
    bool do_provide(ReqB& req, RspB& rsp) override;
};
} // namespace inner

} // namespace app

#endif
