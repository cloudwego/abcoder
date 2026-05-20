#ifndef CN_APP_H
#define CN_APP_H

#include "common.h"

namespace app {

// Same short name "Provider" as common::Provider — must not be misclassified
// as a self-resolved (unresolved) base. app::Provider.Implements should
// contain common::Provider, NOT be empty.
class Provider : public ::common::Provider {
public:
    int provide() override { return 2; }
};

} // namespace app

#endif
