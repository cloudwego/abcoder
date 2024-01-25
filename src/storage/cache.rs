use std::collections::HashMap;
use std::error::Error;
use std::path::Path;

use crate::storage::fs::FileStorage;
use crate::storage::memory::MemoryCache;

pub trait StorageEngine {
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
    let fs = FileStorage::new(Path::new("./tmp_compress")).unwrap();

    let mut cache = CachingStorageEngine::new(mem, fs);
    return Box::new(cache);
}