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

/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pub trait Addable {
    fn id() -> i64;
    fn add(&self, b: i64) -> i64;
}

pub struct AnyInt(i64);

impl AnyInt {
    pub fn id() -> i64 {
        0
    }
    pub fn add(&self, b: i64) -> i64 {
        self.0 + b
    }
}

impl Addable for AnyInt {
    fn add(&self, b: i64) -> i64 {
        // use the method defined in the struct
        self.add(b)
    }
    fn id() -> i64 {
        // use the method defined in the struct
        AnyInt::id()
    }
}

fn add_trait<T: Addable>(i: T, v: i64) {
    println!("{}", i.add(v))
}

#[test]
fn test() {
    add_trait(AnyInt(1), 2);
}
