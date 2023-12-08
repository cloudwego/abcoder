use std::collections::{HashMap, HashSet};
use std::collections::VecDeque;

use crate::compress::parser::{Function, LanguageParser};

pub struct GolangParser {
    functions: HashMap<String, Function>,
    // function name mapped to Function struct
    processed: HashSet<String>,            // names of functions already processed
}

impl LanguageParser for GolangParser {
    fn new() -> Self {
        GolangParser {
            functions: HashMap::new(),
            processed: HashSet::new(),
        }
    }

    fn process(&mut self) {
        // Parse golang project here.
        // This is a rather complex topic and can't be done in a few lines of code.
        // You might want to use some existing libraries for parsing golang code.
    }

    fn summarize(&self, function: &Function) -> String {
        // Summarize the function here.
        // For example, this can output function's name and number of calls.
        let summary = format!(
            "Function name: {}\nNumber of calls: {}",
            function.name,
            function.calls.len()
        );
        summary
    }
}