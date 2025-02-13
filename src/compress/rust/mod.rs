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

use std::{cell::RefCell, collections::HashMap};

pub fn convert_crate(path: &str) -> String {
    // extra last two parts
    let ps: Vec<&str> = path.split("/").collect();
    if ps.len() >= 2 {
        normalize_rust_import(format!("{}#{}", ps[ps.len() - 2], ps[ps.len() - 1]).as_str())
    } else {
        normalize_rust_import(path)
    }
}

pub fn normalize_rust_import(path: &str) -> String {
    let mut normalized = String::new();
    for ch in path.chars() {
        if !ch.is_ascii_alphabetic() && !ch.is_ascii_digit() && ch != '/' {
            normalized += "_";
        } else if ch == '/' {
            normalized.push_str("::");
        } else {
            normalized.push(ch);
        }
    }
    normalized.split("::").fold(String::new(), |acc, p| {
        if acc != "" {
            let p = if let Some(n) = avoid_rust_keywords(p) {
                n
            } else {
                p.to_string()
            };
            format!("{}::{}", acc, p)
        } else {
            p.to_string()
        }
    })
}

pub fn new_rust_impt(_crate: &str, path: &str) -> String {
    let p = if path != "" {
        normalize_rust_import(path)
    } else {
        "*".to_string()
    };
    return format!("{}::{}", normalize_rust_import(_crate), p);
}

static RUST_KEY_WORDS: [&str; 52] = [
    "as", "break", "const", "continue", "crate", "else", "enum", "extern", "false", "fn", "for",
    "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub", "ref", "return",
    "self", "Self", "static", "struct", "super", "trait", "true", "type", "unsafe", "use", "where",
    "while", "abstract", "alignof", "become", "box", "do", "final", "macro", "offsetof",
    "override", "priv", "proc", "pure", "sizeof", "typeof", "unsized", "virtual", "yield",
];

pub fn avoid_rust_keywords(word: &str) -> Option<String> {
    if RUST_KEY_WORDS.contains(&word) {
        Some(format!("r#{}", word))
    } else {
        None
    }
}

// format and replace root
pub fn replace_impt_crate(impt: &str, root: &Option<String>, reop_id: &String) -> String {
    let impt = impt.strip_prefix("use ").unwrap();
    let mut impt = impt.replace(" ", "");
    // for path specify root, replace it
    if let Some(root) = root {
        if impt.starts_with(root) {
            // inside the root, replace it with crate
            impt = impt.replace(root, "crate")
        } else if impt.starts_with("crate::") {
            // outside the root, replace it with repo_id
            impt = impt.replace("crate", reop_id);
        }
    }
    format!("use {}", impt)
}

// rust error format:
/*
error: {{msg}}
  --> {{file:line:col}}
   |
79 |     xxxxxxxxxxxxx
   |         ^^^^^^^^ help: xxxxx
 */
pub fn extract_msg_from_err(err: &str, ignore: bool) -> HashMap<&str, String> {
    let mut files: HashMap<&str, String> = HashMap::new();
    let lines = err.split("\n").collect::<Vec<&str>>();

    let mut i = 0;
    'out: while i < lines.len() {
        let line = lines[i];
        if line.contains("--> ") {
            let file = line
                .trim()
                .strip_prefix("--> ")
                .unwrap()
                .split(":")
                .nth(0)
                .unwrap();
            let msg = lines[i - 1];
            if ignore {
                // ignore specific errors
                for e in RUST_IGNORE_ERRS.iter() {
                    if msg.contains(e) {
                        i += 1;
                        continue 'out;
                    }
                }
            }
            let mut errs = vec![msg, line];
            for j in (i + 1)..lines.len() {
                let line = lines[j];
                if line.contains(" | ") {
                    errs.push(line);
                } else {
                    i = j;
                    break;
                }
            }
            if let Some(old) = files.get_mut(file) {
                old.push_str("\n");
                old.push_str(&errs.join("\n"));
            } else {
                files.insert(file, errs.join("\n"));
            }
        }
        i += 1;
    }
    files
}

static RUST_IGNORE_ERRS: [&str; 5] = [
    "E0425", // not found in crate
    "E0412", // not found in crate
    "E0433", // use of undeclared crate or module
    "E0601", // consider adding a `main` function to
    "E0432", // unresolved import
];

#[derive(Debug, Clone)]
pub struct Cargo {
    pub id: String,
    deps: RefCell<HashMap<String, String>>,
    bins: RefCell<HashMap<String, String>>,
}

impl Cargo {
    pub fn new(id: &str) -> Self {
        Cargo {
            id: normalize_rust_import(id.split("/").last().unwrap_or_default()),
            deps: RefCell::new(HashMap::new()),
            bins: RefCell::new(HashMap::new()),
        }
    }

    pub fn dep(&mut self, deps: &String) {
        let lines = deps.split("\n").collect::<Vec<&str>>();
        for line in lines {
            if !line.contains("=") {
                continue;
            }
            let (a, b) = line.split_once("=").unwrap();
            let name = a.trim();
            let mut version = b.trim();
            if let Some(i) = version.find("//") {
                version = &version[0..i];
            }
            self.deps
                .borrow_mut()
                .insert(name.to_string(), version.to_string());
        }
    }

    pub fn undep(&mut self, name: &String) {
        self.deps.borrow_mut().remove(name);
    }

    pub fn bin(&mut self, name: &String, path: &str) {
        self.bins
            .borrow_mut()
            .insert(name.clone(), path.to_string());
    }

    pub fn to_string(&mut self) -> String {
        let deps = self
            .deps
            .borrow()
            .iter()
            .filter(|(k, _)| !k.contains("crate"))
            .fold(String::new(), |acc, (k, v)| {
                format!("{}\n{} = {}", acc, k, v)
            });
        let bins = self.bins.borrow().iter().fold(String::new(), |acc, b| {
            format!(
                "{}\n[[bin]]\nname = \"{}\"\npath = \"src/{}/main.rs\"\n",
                acc, b.0, b.1
            )
        });
        format!(
            r#"[package]
name = "{}"
version = "0.1.0"
edition = "2021"
{}
[dependencies]
{}
"#,
            self.id, bins, deps,
        )
    }
}
