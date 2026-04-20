from pkg_a import get_symbol_a, SymbolA
from pkg_b import get_symbol_b, SymbolB


def main() -> None:
    a: SymbolA = get_symbol_a()
    b: SymbolB = get_symbol_b(a)
    print(a.get_value())
    print(b.get_value())


if __name__ == "__main__":
    main()
