// Regression scenario for breakpoint A1: external base classes commonly
// define their virtual interface entirely inline in a single header, e.g.
// `cppservice::ApiHandler::process`, `cppservice::Step::process`,
// `cppservice::Strategy::process`. Three different classes, same short
// method name, in the same namespace.
//
// Before A1 fix, clangd reported these as just `process(...)`; the
// collector dropped the class qualifier and collapsed all three into the
// same orphan `ns::process(...)` symbol. After A1, each must appear with
// its own class qualifier and IsMethod=true.
#ifndef SVC_HANDLERS_H
#define SVC_HANDLERS_H

namespace svc {

struct Ctx {
    int v;
};

class ApiHandler {
public:
    // Inline definition: clangd reports sym.Name = "process(Ctx& ctx)".
    int process(Ctx& ctx) {
        ctx.v += 1;
        return ctx.v;
    }
};

class Step {
public:
    // Same short name, different class.
    int process(Ctx& ctx) {
        ctx.v *= 2;
        return ctx.v;
    }
};

class Strategy {
public:
    // Same short name, different class.
    int process(Ctx& ctx) {
        ctx.v -= 3;
        return ctx.v;
    }
};

} // namespace svc

#endif // SVC_HANDLERS_H
