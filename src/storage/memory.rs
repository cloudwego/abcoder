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

use std::collections::HashMap;
use std::error::Error;

use crate::storage::cache::StorageEngine;

pub struct MemoryCache {
    map: HashMap<String, Vec<u8>>,
}

impl MemoryCache {
    pub fn new(cache_size: usize) -> Self {
        MemoryCache { map: HashMap::with_capacity(cache_size) }
    }
}

impl StorageEngine for MemoryCache {
    fn get(&mut self, key: &str) -> Option<Vec<u8>> {
        self.map.get(key).cloned()
    }

    fn put(&mut self, key: &str, value: Vec<u8>) -> Result<(), Box<dyn Error>> {
        self.map.insert(key.to_string(), value);
        Ok(())
    }
}