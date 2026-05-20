#include "concrete.h"

namespace app {

bool ConcreteProvider::do_provide(ReqA& req, RspA& rsp) {
    rsp.v = req.v + 1;
    return true;
}

bool MultiLineProvider::do_provide(ReqA& req, RspA& rsp) {
    rsp.v = req.v + 2;
    return true;
}

} // namespace app

namespace app {
namespace inner {
bool BareNameProvider::do_provide(ReqB& req, RspB& rsp) {
    rsp.v = req.v + 3;
    return true;
}
} // namespace inner

} // namespace app
