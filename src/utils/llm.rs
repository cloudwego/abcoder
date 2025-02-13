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
