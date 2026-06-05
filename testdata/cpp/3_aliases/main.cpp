#include "types.h"
#include <iostream>

int main() {
    aliases::MyInt a = 1;
    aliases::MyDouble d = 2.5;
    aliases::Cnt c = aliases::make_cnt();
    aliases::CntAlias ca = aliases::make_via_using();
    c.bump();
    ca.bump();
    std::cout << aliases::produce(a) << " " << d << " " << c.n << " " << ca.n << "\n";
    return 0;
}
