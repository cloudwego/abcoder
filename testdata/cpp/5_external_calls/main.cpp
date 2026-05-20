// Regression scenario for breakpoint A2: in default mode (no
// --load-external-symbol), calls to functions/methods defined outside the
// workspace would previously get dropped from MethodCalls/FunctionCalls,
// because exportSymbol returns ErrExternalSymbol for the dep and the
// caller silently `continue`d.
//
// After A2's lightIdentityForExternal fallback, the edge is preserved
// using a best-effort name+module Identity, so the call graph can still
// be walked across the workspace boundary.
//
// This file exercises the path by calling a couple of std/external APIs.
#include <cstdio>
#include <string>

namespace probe {

// Free function calling external std::printf.
int run_printf(int x) {
    return std::printf("v=%d\n", x);
}

// Method calling external std::string::size().
class Probe {
public:
    explicit Probe(std::string s) : s_(std::move(s)) {}
    std::size_t length() const {
        return s_.size();
    }
private:
    std::string s_;
};

} // namespace probe

int main() {
    probe::Probe p("hi");
    return static_cast<int>(p.length()) + probe::run_printf(7);
}
