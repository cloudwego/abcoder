#include "concrete.h"

int main(int argc, char** argv) {
    app::ReqA req{1};
    app::RspA rsp{0};
    app::ConcreteProvider p;
    // Template function call — should produce an FC edge to common::run.
    bool ok = common::run<app::ConcreteProvider, app::RspA>(argc, argv);
    return (ok && p.provide(req, rsp)) ? 0 : 1;
}
