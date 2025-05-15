from dataclasses import dataclass


@dataclass
class IntPair:
    a: int
    b: int

    def sum(self):
        return self.a + self.b


def main() -> None:
    my_pair = IntPair(a=10, b=20)
    print(f"Original pair: {my_pair}")
