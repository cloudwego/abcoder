#include "types.h"

int main() {
    kc::Holder h;
    return h.handle(1) + h.handle(2L);
}
