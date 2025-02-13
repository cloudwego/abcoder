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

use std::fmt;
use std::fs;
use std::fs::File;
use std::io::{self, BufRead};
use std::path::Path;
use std::path::PathBuf;

use crate::config::CONFIG;

pub fn read_file(path: &str) -> io::Result<String> {
    fs::read_to_string(path)
}

// Function to count the lines in a file
pub fn count_lines(path: &Path, skip_empty: bool) -> io::Result<usize> {
    let file = File::open(path)?;
    let reader = io::BufReader::new(file);

    if skip_empty {
        let line_count = reader
            .lines()
            .filter_map(Result::ok)
            .filter(|line| !line.trim().is_empty())
            .count();
        return Ok(line_count);
    }

    let line_count = reader.lines().count();
    Ok(line_count)
}

// Define our tree node
pub struct Node {
    name: String,
    children: Vec<Node>,
}

impl Node {
    fn new(name: String) -> Node {
        Node {
            name,
            children: Vec::new(),
        }
    }

    fn add_child(&mut self, child: Node) {
        self.children.push(child);
    }
}

// Use this to print the tree with indentation
impl fmt::Display for Node {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        fn recurse(node: &Node, f: &mut fmt::Formatter<'_>, depth: usize) -> fmt::Result {
            for _ in 0..depth {
                write!(f, "   ")?;
            }
            let display_name = node.name.trim_start_matches(&CONFIG.repo_dir);
            writeln!(f, "└──{}", display_name)?;

            for child in &node.children {
                recurse(child, f, depth + 1)?;
            }

            Ok(())
        }

        recurse(self, f, 0)
    }
}

pub fn tree(dir: &PathBuf, suffix: &str) -> Option<Node> {
    let mut node = Node::new(dir.to_string_lossy().to_string());
    let mut has_matching_file = false;

    if let Ok(entries) = fs::read_dir(dir) {
        for entry in entries {
            if let Ok(entry) = entry {
                let path = entry.path();
                if path.is_dir() {
                    if let Some(child) = tree(&path, suffix) {
                        has_matching_file = true;
                        node.add_child(child);
                    }
                } else {
                    if let Some(ext) = path.extension() {
                        if ext.to_str().unwrap() == suffix {
                            has_matching_file = true;
                            node.add_child(Node::new(path.display().to_string()));
                        }
                    }
                }
            }
        }
    }

    if has_matching_file {
        Some(node)
    } else {
        None
    }
}

pub fn camel_to_snake(camel: &str) -> String {
    let mut snake = String::new();
    for (i, ch) in camel.chars().enumerate() {
        if ch.is_ascii_uppercase() && i > 0 {
            // get i-1th char
            let before_ch = camel.chars().nth(i - 1).unwrap();
            let after_ch = if i < camel.len() - 1 {
                camel.chars().nth(i + 1).unwrap()
            } else {
                ' '
            };
            if before_ch.is_alphanumeric()
                && (before_ch.is_ascii_lowercase() || before_ch.is_digit(10))
                || (after_ch.is_alphanumeric() && after_ch.is_ascii_lowercase()
                    || before_ch.is_digit(10))
            {
                snake.push('_');
            }
        }
        snake.push(ch.to_ascii_lowercase());
    }
    snake
}

pub fn snake_to_camel(snake: &str) -> String {
    let mut camel = String::new();
    let mut upper = true;
    for ch in snake.chars() {
        if ch == '_' {
            upper = true;
        } else {
            if upper {
                camel.push(ch.to_ascii_uppercase());
                upper = false;
            } else {
                camel.push(ch);
            }
        }
    }
    camel
}

#[test]
fn test_camel_to_snake() {
    assert_eq!(camel_to_snake("camel"), "camel");
    assert_eq!(camel_to_snake("Camel"), "camel");
    assert_eq!(camel_to_snake("CAMEL"), "camel");
    assert_eq!(camel_to_snake("JSONString"), "json_string");
    assert_eq!(camel_to_snake("CamelCase"), "camel_case");
    assert_eq!(camel_to_snake("Camel2Case"), "camel2_case");
    assert_eq!(camel_to_snake("Camel2Case3A"), "camel2_case3_a");
    assert_eq!(camel_to_snake("CamelJSONString"), "camel_json_string");
}

#[test]
fn test_snake_camel() {
    assert_eq!(snake_to_camel("camel"), "Camel");
    assert_eq!(snake_to_camel("_camel"), "Camel");
}
