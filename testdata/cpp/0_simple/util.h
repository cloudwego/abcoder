#ifndef UTIL_H
#define UTIL_H

#include <string>

namespace util {

// ===== 全局变量（声明）=====
extern int g_counter;
extern const std::string g_appName;

// ===== 类声明 =====
class Greeter {
public:
    explicit Greeter(std::string prefix);

    std::string greet(const std::string& name) const;
    void bump();
    int localCount() const;

private:
    std::string prefix_;
    int localCount_{0};
};

} // namespace util

#endif // UTIL_H
