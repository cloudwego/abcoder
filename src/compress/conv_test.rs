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

// Copyright 2024 CloudWeGo Authors
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

use crate::compress::types::types::Identity;

use super::{
    conv,
    types::types::{Node, NodeKind},
};
use std::{
    collections::{HashMap, HashSet},
    fs,
};

#[test]
fn test_collect_rust_item_macro() {
    let codes = fs::read_to_string("testdata/rust/main.rs").unwrap();
    let mut imps = HashSet::new();
    let mut sub = "".to_string();
    let node: Node = Node {
        kind: NodeKind::Var,
        id: Identity {
            mod_path: "mod/x".to_string(),
            pkg_path: "mod/x/pkg".to_string(),
            name: "GlobalVar".to_string(),
        },
        dependencies: Vec::new(),
        references: Vec::new(),
    };
    let rust_id = "GLOBAL_VAR".to_string();
    let rust_type = None;
    let go_new_type = None;
    let kind = NodeKind::Var;

    // parse return Rust codes to AST
    match syn::parse_file(codes.as_str()) {
        Ok(file) => {
            // find id item
            for item in &file.items {
                // println!("item: {}", quote::quote! {#item});
                let (s, found) = conv::collect_rust_item(
                    item,
                    &node,
                    &rust_id,
                    &rust_type,
                    &go_new_type,
                    &kind,
                    &mut imps,
                );
                if found {
                    sub = s;
                    break;
                }
            }
        }
        Err(e) => {
            println!(
                "[WARNING] fail parse rust codes `{}`, err: {}",
                codes,
                e.to_string()
            );
        }
    }
    assert_eq!(
        sub,
        "lazy_static ! { pub static ref GLOBAL_VAR : Mutex < i64 > = Mutex :: new (0) ; }"
    );
}
