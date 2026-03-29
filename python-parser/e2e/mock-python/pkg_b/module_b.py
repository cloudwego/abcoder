from dataclasses import dataclass
from pkg_a.module_a import SymbolA


@dataclass(frozen=True)
class SymbolB:
    """强类型的 Symbol B"""
    value: str

    def get_value(self) -> str:
        return self.value


def symbol_b(a: SymbolA) -> SymbolB:
    result: str = a.get_value()
    return SymbolB(value=f"Symbol B uses: {result}")
