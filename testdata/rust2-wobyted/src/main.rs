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

mod entity;

static MY_INT2: u32 = 42;

fn main() {
    let a = entity::MyStruct::new(1, entity::MyInt2(2));
    println!("{} + {:?} = {}", a.a, a.b, a.add());
    let b = entity::add(1, entity::MyInt2(2));
    println!("1 + 2 = {}", b);
    println!("MY_INT = {}", entity::MY_INT);
    println!("MY_INT2 = {}", MY_INT2);

    _ = entity::MyEnum::A(1);
    _ = entity::MyEnum::B(entity::MyInt2(2));
    _ = crate::my_macro!(1);

    let mut buf = Vec::new();
    entity::func::write_to_output(&mut buf, &a, &a).unwrap();
}
