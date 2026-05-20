#ifndef PROVIDER_H
#define PROVIDER_H

// Mirrors the freq_service pattern: a templated NVI base class.
//   Provider<Req,Rsp>::provide()  — non-virtual, calls do_provide()
//   do_provide()                  — pure virtual; derived overrides
namespace common {

template <class Req, class Rsp>
class Provider {
public:
    // Header-inline body — abcoder must still record the do_provide()
    // call edge so synthesized inherited methods inherit it.
    bool provide(Req& req, Rsp& rsp) { return do_provide(req, rsp); }

protected:
    virtual bool do_provide(Req& req, Rsp& rsp) = 0;
};

// Templated free function — mirrors freq_service's main()'s call to
// `cppservice::main<...>(argc, argv)`. The FC edge from a caller to
// this kind of call must surface as a FunctionCall.
template <class Handler, class Mgr>
bool run(int argc, char** argv);

template <class Handler, class Mgr>
bool run(int argc, char** argv) {
    (void)argc; (void)argv;
    return true;
}

// Strategy-style class: only declares `process`, the definition would
// normally live in a .cpp file not present in our workspace. Mirrors
// freq_service's `cppservice::Strategy::process` (problem #2 in
// docs/cpp_known_issues.md): the AST must still carry the method
// node even when no body exists.
class Strategy {
public:
    virtual bool process(int& ctx);
};

} // namespace common

#endif
