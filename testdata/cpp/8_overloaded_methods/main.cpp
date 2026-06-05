#include "overloads.h"

int main() {
    ovl::Derived d;
    int a = d.foo(1);
    int b = d.foo(std::string("x"));
    return a + b;
}
