#include "shapes.h"
#include <iostream>

int main() {
    shapes::Circle c(2.0);
    shapes::Square s(3.0);
    shapes::LabeledCircle lc(1.0, "label");
    shapes::IntStore is;

    std::cout << c.area() << "\n";
    std::cout << s.area() << "\n";
    std::cout << lc.area() << " " << lc.label() << "\n";
    is.inc();
    std::cout << is.get() << "\n";
    return 0;
}
