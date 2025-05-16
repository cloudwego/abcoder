# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from dataclasses import dataclass
from typing import Union


@dataclass
class IntPair:
    a: int
    b: int


def swap_pair(pair: IntPair) -> None:
    """
    Swaps the values of a and b in an IntPair.
    Note: The original Rust code had a logical error if a swap was intended;
    it would result in both pair.a and pair.b being set to the original value of pair.a.
    This Python version implements a correct swap.
    """
    pair.a, pair.b = pair.b, pair.a


class IntVariant:
    def __init__(self, value: int):
        self.value: int = value

    def __repr__(self) -> str:
        return f"IntVariant({self.value})"


class CharVariant:
    def __init__(self, value: int):
        if not (0 <= value <= 255):
            raise ValueError(
                "CharVariant value must be an integer between 0 and 255 (u8 equivalent)"
            )
        self.value: int = value

    def __repr__(self) -> str:
        return f"CharVariant(value={self.value}, char='{chr(self.value)}')"


IntOrChar = Union[IntVariant, CharVariant]


def add(a: int, b: int) -> int:
    return a + b


def compare(a: int, b: int) -> int:
    if a < b:
        return -1
    elif a > b:
        return 1
    else:
        return 0


def main() -> None:
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
