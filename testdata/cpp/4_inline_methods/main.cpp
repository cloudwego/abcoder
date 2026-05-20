#include "handlers.h"

int main() {
    svc::Ctx c{1};
    svc::ApiHandler a;
    svc::Step s;
    svc::Strategy g;
    int x = a.process(c) + s.process(c) + g.process(c);
    return x;
}
