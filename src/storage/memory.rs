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