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

use std::io::{self, Write};

use super::{MyEnum, MyInt, MyInt2, MyStruct, MyTrait};

#[warn(unused_variables)]
pub fn write_to_output<W: Write, T: MyTrait>(
    output: &mut W,
    input: &T,
    arg: &MyStruct,
) -> io::Result<(MyEnum, MyInt2)> {
    Ok((MyEnum::A(input.my_trait() + arg.a), arg.b))
}

use core::f64::consts::E;
use std::f32::consts::PI;

fn apply_closure<F, Y: Fn(MyInt) -> MyInt2>(x: Option<MyInt>, func: F, func2: Y) -> MyInt2
where
    F: Fn(MyInt) -> MyInt2,
{
    func(x.unwrap())
}

/// This is a test function
/// # Example
/// ```
/// let obj = new_obj();
/// ```
pub fn new_obj() -> MyStruct {
    MyStruct { a: 1, b: MyInt2(2) }
}
