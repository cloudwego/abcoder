#include "util.h"
#include <utility>  // std::move

namespace util {

// ===== 全局变量（定义，只能在一个 .cpp 中定义）=====
int g_counter = 0;
const std::string g_appName = "DemoApp";

// ===== 类实现 =====
Greeter::Greeter(std::string prefix) : prefix_(std::move(prefix)) {}

std::string Greeter::greet(const std::string& name) const {
    return prefix_ + ", " + name + "!";
}

void Greeter::bump() {
    ++g_counter;
    ++localCount_;
}

int Greeter::localCount() const {
    return localCount_;
}

} // namespace util
