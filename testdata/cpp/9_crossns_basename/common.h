#ifndef CN_COMMON_H
#define CN_COMMON_H

namespace common {

class Provider {
public:
    virtual ~Provider() = default;
    virtual int provide() { return 1; }
};

} // namespace common

#endif
