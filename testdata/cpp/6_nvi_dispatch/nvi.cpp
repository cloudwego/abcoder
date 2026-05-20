#include "nvi.h"

namespace nvi {

bool Provider::provide(Ctx& c) {
    return do_provide(c);
}

bool Provider::do_provide(Ctx& c) {
    return c.v > 0;
}

bool RealProvider::do_provide(Ctx& c) {
    c.v += 1;
    return c.v > 0;
}

bool Caller::drive(Ctx& c) {
    return real_.provide(c);
}

} // namespace nvi
