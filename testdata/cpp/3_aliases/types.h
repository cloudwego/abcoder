#ifndef ALIASES_TYPES_H
#define ALIASES_TYPES_H

#include <string>

namespace aliases {

// Real type that other declarations refer to.
struct Counter {
    int n{0};
    void bump() { ++n; }
};

// typedef on a primitive — should be EMITTED as Type with TypeKind=typedef.
typedef int MyInt;

// typedef on a user-defined type — same: kept, TypeKind=typedef.
typedef Counter Cnt;

// using-alias on a primitive — should be DROPPED (not emitted), references
// to MyDouble should resolve directly to `double`.
using MyDouble = double;

// using-alias on a user-defined type — should be DROPPED, references to
// CntAlias should resolve to Counter.
using CntAlias = Counter;

// A function that exercises both typedef and using-alias parameters/returns.
inline MyInt produce(MyInt x) { return x + 1; }
inline Cnt make_cnt() { return Cnt{}; }
inline CntAlias make_via_using() { return CntAlias{}; }

} // namespace aliases

#endif // ALIASES_TYPES_H
