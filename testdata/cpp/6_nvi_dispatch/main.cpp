#include "nvi.h"

int main() {
    nvi::Ctx c{0};
    nvi::Caller k;
    return k.drive(c) ? 0 : 1;
}
