use std::error::Error;
use std::fs::File;
use std::io::{Read, Write};
use std::path::{Path, PathBuf};

use crate::storage::cache::StorageEngine;

pub struct FileStorage {
    base_dir: PathBuf,
}

impl FileStorage {
    pub fn new(base_dir: &Path) -> Result<Self, Box<dyn Error>> {
        if !base_dir.exists() {
            std::fs::create_dir_all(base_dir)?;
        }
        Ok(FileStorage { base_dir: base_dir.to_path_buf() })
    }
}


impl StorageEngine for FileStorage {
    fn get(&mut self, key: &str) -> Option<Vec<u8>> {
        let mut path = self.base_dir.clone();
        path.push(key);
        let mut file = match File::open(path) {
            Ok(file) => file,
            Err(_) => return None,
        };
        let mut contents = Vec::new();
        match file.read_to_end(&mut contents) {
            Ok(_) => Some(contents),
            Err(_) => None,
        }
    }

    fn put(&mut self, key: &str, value: Vec<u8>) -> Result<(), Box<dyn Error>> {
        let mut path = self.base_dir.clone();
        path.push(key);
        let mut file = File::create(path)?;
        file.write_all(&value)?;
        Ok(())
    }
}