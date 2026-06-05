#ifndef UD_DERIVED_H
#define UD_DERIVED_H

#include "base.h"

namespace ud {

class Derived : public Base {
public:
    // using-declaration (NOT a type alias): re-expose Base::foo overloads
    // so they're visible after Derived hides them with its own foo. This
    // is C++17 §10.3.3 — name binding, not type aliasing.
    using Base::foo;

    int foo(const char* s) { return s[0]; }
};

} // namespace ud

#endif
