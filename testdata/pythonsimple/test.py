from typing import Union
from test2 import IntPair
from test3 import *


def swap_pair(pair: IntPair) -> None:
    """
    Swaps the values of a and b in an IntPair.
    Note: The original Rust code had a logical error if a swap was intended;
    it would result in both pair.a and pair.b being set to the original value of pair.a.
    This Python version implements a correct swap.
    """
    pair.a, pair.b = pair.b, pair.a


from test3 import *


def add(a: int, b: int) -> int:
    return a + b


def compare(a: int, b: int) -> int:
    if a < b:
        return -1
    elif a > b:
        return 1
    else:
        return 0


IntOrChar = Union[IntVariant, CharVariant]


def main() -> None:
    ls = list((1, 2))

    x = add(2, 3)
    print(x)

    my_pair = IntPair(a=10, b=20)
    print(f"Original pair: {my_pair}")
    swap_pair(my_pair)
    print(f"Swapped pair: {my_pair}")

    val1: IntOrChar = IntVariant(123)
    val2: IntOrChar = CharVariant(ord("A"))

    print(f"IntOrChar 1: {val1}")
    print(f"IntOrChar 2: {val2}")

    if isinstance(val1, IntVariant):
        print(f"val1 is an IntVariant with value: {val1.value}")
    if isinstance(val2, CharVariant):
        print(
            f"val2 is a CharVariant with u8 value: {val2.value} (char: '{chr(val2.value)}')"
        )

    print(f"Comparing 5 and 10: {compare(5, 10)}")
    print(f"Comparing 10 and 5: {compare(10, 5)}")
    print(f"Comparing 7 and 7: {compare(7, 7)}")


if __name__ == "__main__":
    main()
