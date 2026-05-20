// Forward-declare a template base in two distinct namespaces, both named
// `Provider`. The base bodies are NOT defined here — clangd cannot resolve
// them, so collect.go will route through the cppUnresolvedBases path.
// After the fix, app::DerivedA.Implements should record
// `third_party::Provider` (with namespace) and app::DerivedB.Implements
// should record `other_pkg::Provider`. Today both collapse to bare
// `Provider`, making them indistinguishable.

namespace third_party {
template <class Req, class Rsp>
class Provider;
}

namespace other_pkg {
template <class T>
class Provider;
}

namespace app {

class DerivedA : public ::third_party::Provider<int, int> {};
class DerivedB : public ::other_pkg::Provider<int> {};

} // namespace app

int main() { return 0; }
