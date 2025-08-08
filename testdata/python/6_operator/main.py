# Copyright 2025 CloudWeGo Authors
#
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


class A:
    def __init__(self):
        self.value = 10

    def get_value(self):
        return self.value

    def __add__(self, other):
        if isinstance(other, A):
            return A(self.value + other.value)
        return NotImplemented


def main():
    a1 = A()
    a2 = A()

    print("Value of a1:", a1.get_value())
    print("Value of a2:", a2.get_value())

    # There should be a dependency from main to A.__add__
    a3 = a1 + a2
    print("Value of a3 (a1 + a2):", a3.get_value())
