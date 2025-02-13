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

use serde::{Deserialize, Serialize};


#[derive(Serialize, Deserialize, Debug)]
pub struct Function {
    pub(crate) name: String,
    pub(crate) is_method: bool,
    pub(crate) is_third_party: bool,
    pub(crate) associated_struct: String,
    pub(crate) content: String,
    pub(crate) compressed_content: CompressedFunction,
    pub(crate) calls: Vec<String>,  // names of the functions this one calls
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CompressedFunction {
    pub(crate) function_name: String,
    pub(crate) description: String,
    pub(crate) input: String,
    pub(crate) output: String,
    pub(crate) side_effect: String,
}

pub trait LanguageParser {
    fn new() -> Self;
    fn process(&mut self);
    fn summarize(&self, _function: &Function) -> String;
}