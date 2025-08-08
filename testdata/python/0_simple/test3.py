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
