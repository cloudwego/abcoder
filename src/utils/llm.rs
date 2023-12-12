use std::fs::File;
use std::io::{self, BufRead};
use std::path::Path;

// roughly count the token count in a file.
pub fn count_tokens_rough(path: &Path) -> io::Result<usize> {
    let file = File::open(path)?;
    let reader = io::BufReader::new(file);

    let mut token_count = 0;

    for line in reader.lines() {
        let line = line?;
        let tokens: Vec<&str> = line
            .split(|c: char| c.is_whitespace() || c.is_ascii_punctuation())
            .collect();
        token_count += tokens.len();
    }

    Ok(token_count)
}
