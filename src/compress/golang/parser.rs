use std::collections::{HashMap, HashSet};
use crate::compress::process::Function;
use super::super::process::LanguageParser;

struct GolangParser {
    functions: HashMap<String, Function>,
    processed: HashSet<String>,
}

impl LanguageParser for GolangParser {
    fn new() -> Self {
        todo!()
    }

    fn add_function(&mut self, name: String, content: String, calls: Vec<String>) {
        todo!()
    }

    fn process_functions(&mut self) {
        todo!()
    }

    fn process_function(&mut self, function: &Function) {
        todo!()
    }

    fn summarize(&self, _function: &Function) -> String {
        todo!()
    }
    // TODO: implement all the methods for GolangParser
}