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
use std::fs::File;
use std::io::{Read, Write};
use std::ops::Add;
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
        Ok(FileStorage {
            base_dir: base_dir.to_path_buf(),
        })
    }
}

impl StorageEngine for FileStorage {
    fn get(&mut self, key: &str) -> Option<Vec<u8>> {
        let mut path = self.base_dir.clone();
        let safe_key = key.replace("/", "_");
        let json_key = safe_key.add(".json");
        path.push(json_key);
        println!("read file cached: {:?}", path);
        let mut file = match File::open(&path) {
            Ok(file) => file,
            Err(_) => {
                println!("not cached: {:?}", path);
                return None;
            }
        };
        let mut contents = Vec::new();
        match file.read_to_end(&mut contents) {
            Ok(_) => Some(contents),
            Err(_) => None,
        }
    }

    fn put(&mut self, key: &str, value: Vec<u8>) -> Result<(), Box<dyn Error>> {
        let mut path = self.base_dir.clone();
        let safe_key = key.replace("/", "_");
        let json_key = safe_key.add(".json");
        path.push(json_key);
        let mut file = File::create(path)?;
        file.write_all(&value)?;
        Ok(())
    }
}
