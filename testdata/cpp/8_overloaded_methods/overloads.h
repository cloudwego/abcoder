#ifndef OVL_OVERLOADS_H
#define OVL_OVERLOADS_H

#include <string>

namespace ovl {

class Base {
public:
    // Two overloads, same short name "foo", different signatures.
    virtual int foo(int x) { return x + 1; }
    virtual int foo(const std::string& s) { return (int)s.size(); }
};

class Derived : public Base {
public:
    // Override only foo(int). foo(const std::string&) must still be
    // inherited via NVI synthesis, NOT collapsed by short-name dedup.
    int foo(int x) override { return x + 100; }
};

} // namespace ovl

#endif
