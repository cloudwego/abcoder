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

pub type MyInt = i32;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MyInt2(pub i32);

impl std::ops::Add<MyInt> for MyInt2 {
    type Output = MyInt;

    fn add(self, rhs: MyInt) -> MyInt {
        self.0 + rhs
    }
}

pub const MY_INT: MyInt = 42;

pub fn add(a: MyInt, b: MyInt2) -> MyInt {
    b + a
}

pub struct MyStruct {
    pub a: MyInt,
    pub b: MyInt2,
}

impl MyStruct {
    pub fn new(a: MyInt, b: MyInt2) -> MyStruct {
        MyStruct { a, b }
    }

    pub fn add(&self) -> MyInt {
        self.b + self.a
    }
}

trait MyTrait {
    fn my_trait(&self) -> MyInt;
}

impl MyTrait for MyStruct {
    fn my_trait(&self) -> MyInt {
        self.a
    }
}

static MY_STATIC: MyInt = 42;

use lazy_static::lazy_static;

lazy_static! {
    pub static ref MY_LAZY_STATIC: MyInt = 44;
}

#[derive(Debug)]
pub enum MyEnum {
    A(MyInt),
    B(MyInt2),
}

#[macro_export]
macro_rules! my_macro {
    ($e:expr) => {
        $e
    };
}

pub mod func;
pub mod inter;

lazy_static! {
    pub static ref MY_LAZY_STATIC2: MyInt = 45;
}
