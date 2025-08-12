// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <stdio.h>

#include "pair.h"

union IntOrChar {
  int i;
  char c;
};

extern int add(int, int);

#define MAXN 100
int arr[MAXN];

int compare(const void *a, const void *b) {
  int int_a = *((int *)a);
  int int_b = *((int *)b);
  if (int_a < int_b)
    return -1;
  if (int_a > int_b)
    return 1;
  return 0;
}

int main() {
  StructIntPair x;
  x.a = 5;
  x.b = 6;
  swapPair(&x);
  struct IntPair y = myself(&x);
  return y.a + y.b;
}
