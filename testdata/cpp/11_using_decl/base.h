#ifndef UD_BASE_H
#define UD_BASE_H

namespace ud {
class Base {
public:
    int foo(int x) { return x + 1; }
    int foo(double x) { return (int)x + 2; }
};
} // namespace ud

#endif
