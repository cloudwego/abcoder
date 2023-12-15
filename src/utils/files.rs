use std::fmt;
use std::fs;
use std::fs::File;
use std::io::{self, BufRead};
use std::path::Path;
use std::path::PathBuf;

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
            let display_name = node.name.trim_start_matches("./tmp/");
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
