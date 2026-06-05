#include "derived.h"

int main() {
    ud::Derived d;
    // All three should resolve: Base::foo(int), Base::foo(double), Derived::foo(const char*).
    return d.foo(1) + d.foo(2.0) + d.foo("x");
}
