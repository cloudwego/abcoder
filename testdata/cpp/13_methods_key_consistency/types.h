#ifndef KC_TYPES_H
#define KC_TYPES_H

namespace kc {

// Two native overloads on the same class. Type.Methods is keyed by short
// name (cppBaseName) so both `handle(int)` and `handle(long)` collide
// into a single map entry "handle" — the second iteration silently
// overwrites the first.
class Holder {
public:
    int handle(int x) { return x + 1; }
    int handle(long x) { return (int)x + 2; }
};

} // namespace kc

#endif
