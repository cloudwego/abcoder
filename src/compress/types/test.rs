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

use std::{cell::RefCell, collections::HashMap, rc::Rc};

fn ensure_node_in_map<'a>(id: i32, map: &'a mut HashMap<i32, i32>) -> &'a mut i32 {
    map.entry(id).or_insert(0)
}

fn test_multi_mut(data: HashMap<i32, Vec<i32>>) {
    let graph = Rc::new(RefCell::new(HashMap::new()));

    for (k, _mod) in data {
        // First, get the value of `k` from the graph
        let mut inode_value = {
            let mut graph_ref = graph.borrow_mut();
            let inode = ensure_node_in_map(k, &mut graph_ref);
            *inode
        };

        for id in _mod {
            // Update the value of `id` in the graph with the value of `k`
            let mut graph_ref = graph.borrow_mut();
            let v = ensure_node_in_map(id, &mut graph_ref);
            *v = inode_value;
            inode_value += 1;
        }

        // update to k
        {
            let mut graph_ref = graph.borrow_mut();
            let inode = ensure_node_in_map(k, &mut graph_ref);
            *inode = inode_value;
        }
    }
}

#[test]
fn test_main() {
    let data = vec![(1, vec![2, 3]), (2, vec![3, 4])];
    let mut map = HashMap::new();
    for (k, v) in data {
        let mut vec = Vec::new();
        for i in v {
            vec.push(i);
        }
        map.insert(k, vec);
    }
    test_multi_mut(map);
}
