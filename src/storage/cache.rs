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

use std::error::Error;
use std::path::Path;

use crate::compress::compress;
use crate::compress::types::types::Repository;
use crate::config::CONFIG;
use crate::storage::fs::FileStorage;
use crate::storage::memory::MemoryCache;

pub trait StorageEngine: Send {
    fn get(&mut self, key: &str) -> Option<Vec<u8>>;
    fn put(&mut self, key: &str, value: Vec<u8>) -> Result<(), Box<dyn Error>>;
}

pub struct CachingStorageEngine<C: StorageEngine, B: StorageEngine> {
    cache: C,
    backend: B,
}

impl<C: StorageEngine, B: StorageEngine> CachingStorageEngine<C, B> {
    pub fn new(cache: C, backend: B) -> Self {
        CachingStorageEngine { cache, backend }
    }
}

impl<C: StorageEngine, B: StorageEngine> StorageEngine for CachingStorageEngine<C, B> {
    fn get(&mut self, key: &str) -> Option<Vec<u8>> {
        if let Some(value) = self.cache.get(key) {
            Some(value)
        } else {
            if let Some(value) = self.backend.get(key) {
                let _ = self.cache.put(key, value.clone());
                Some(value)
            } else {
                None
            }
        }
    }

    fn put(&mut self, key: &str, value: Vec<u8>) -> Result<(), Box<dyn Error>> {
        self.cache.put(key, value.clone())?;
        self.backend.put(key, value)
    }
}

pub fn get_cache() -> Box<dyn StorageEngine> {
    let mem = MemoryCache::new(12);
    let fs = FileStorage::new(Path::new(CONFIG.cache_dir.as_str())).unwrap();

    let mut cache = CachingStorageEngine::new(mem, fs);
    return Box::new(cache);
}

pub fn load_repo(cache: &mut Box<dyn StorageEngine>, repo_name: &str) -> Option<Box<Repository>> {
    if let Some(repo) = cache.get(repo_name) {
        if let Ok(repo) = String::from_utf8(repo) {
            if let Ok(repo) = compress::from_json(repo_name, &repo) {
                let repo = Box::new(repo);
                return Some(repo);
            }
        }
    }

    None
}
