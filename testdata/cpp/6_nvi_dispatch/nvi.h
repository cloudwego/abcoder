// Regression scenario for breakpoint C — Model B (synthesized inherited
// methods + devirtualization on this-calls).
//
// `Provider::provide` is a non-virtual interface that calls the virtual
// `do_provide`. A derived class `RealProvider` overrides `do_provide`
// only. Without Model B, the call graph dead-ends at `Provider::provide`
// (external body) and never reaches `RealProvider::do_provide`.
//
// After Model B, an inherited `RealProvider::provide` is synthesized
// with the base's body and its `this`-call to `do_provide` rewritten to
// `RealProvider::do_provide`.
//
// Implementations live in nvi.cpp (out-of-line) so clangd returns full
// semantic tokens for them regardless of preamble caching state.
#ifndef NVI_H
#define NVI_H

namespace nvi {

struct Ctx {
    int v;
};

class Provider {
public:
    bool provide(Ctx& c);

protected:
    virtual bool do_provide(Ctx& c);
};

class RealProvider : public Provider {
protected:
    bool do_provide(Ctx& c) override;
};

class Caller {
public:
    bool drive(Ctx& c);

private:
    RealProvider real_;
};

} // namespace nvi

#endif // NVI_H
