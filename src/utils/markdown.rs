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

use std::fs;
use std::path::Path;

use serde::{Deserialize, Serialize};

use crate::utils::files;

pub fn get_all_md(dir: &Path, files_list: &mut Vec<String>) -> std::io::Result<()> {
    if dir.is_dir() {
        for entry in fs::read_dir(dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_dir() && !path.ends_with(".github") {
                get_all_md(&path, files_list)?;
            } else if path.is_file() {
                if let Some(ext) = path.extension() {
                    if ext == "md" {
                        files_list.push(path.to_string_lossy().to_string());
                    }
                }
            }
        }
    }
    Ok(())
}

#[derive(Serialize, Deserialize, Debug)]
pub struct FileWithContent {
    pub(crate) name: String,
    pub(crate) content: String,
}

pub fn get_readme_json(repo_dir: &Path) -> Option<String> {
    let mut md_list = Vec::new();
    get_all_md(repo_dir, &mut md_list).expect("TODO: panic message");
    for md in &md_list {
        if md.ends_with("README.md") {
            let content = files::read_file(md).unwrap();
            let readme = &FileWithContent {
                name: "README".to_string(),
                content,
            };

            return Some(serde_json::to_string_pretty(readme).unwrap());
        }
    }
    None
}
