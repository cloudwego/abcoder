from dataclasses import dataclass


@dataclass(frozen=True)
class SymbolA:
    """强类型的 Symbol A"""
    value: str

    def get_value(self) -> str:
        return self.value


def symbol_a() -> SymbolA:
    return SymbolA(value="Symbol A from pkg_a")
