#include "util.h"
#include <iostream>

int main() {
    std::cout << "app=" << util::g_appName << "\n";
    std::cout << "g_counter=" << util::g_counter << "\n";

    util::Greeter hi("Hello");
    std::cout << hi.greet("Alice") << "\n";

    hi.bump();
    hi.bump();

    std::cout << "g_counter=" << util::g_counter << "\n";
    std::cout << "hi.localCount()=" << hi.localCount() << "\n";
    return 0;
}
