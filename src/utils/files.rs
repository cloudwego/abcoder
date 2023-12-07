use std::fs::File;
use std::io::{self, BufRead};
use std::path::Path;

// Function to count the lines in a file
pub fn count_lines(path: &Path, skip_empty: bool) -> io::Result<usize> {
    let file = File::open(path)?;
    let reader = io::BufReader::new(file);

    if skip_empty {
        let line_count = reader.lines()
            .filter_map(Result::ok)
            .filter(|line| !line.trim().is_empty())
            .count();
        return Ok(line_count);
    }

    let line_count = reader.lines().count();
    Ok(line_count)
}