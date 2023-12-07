use std::fs::File;
use std::io::{self, BufRead};
use std::path::Path;

// Function to count the lines in a file
pub fn count_lines(path: &Path) -> io::Result<usize> {
    let file = File::open(path)?;
    let reader = io::BufReader::new(file);

    let line_count = reader.lines().count();
    Ok(line_count)
}