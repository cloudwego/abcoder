use std::collections::{HashMap, HashSet};

use crate::compress::parser::Function;

use super::super::parser::LanguageParser;

struct GolangParser {
    functions: HashMap<String, Function>,
    processed: HashSet<String>,
}

impl LanguageParser for GolangParser {
    fn new() -> Self {
        Self {
            functions: HashMap::new(),
            processed: HashSet::new(),
        }
    }

    fn add_function(&mut self, name: String, content: String, calls: Vec<String>) {
        let function = Function { name: name.clone(), content, calls };
        self.functions.insert(name, function);
    }

    fn process_functions(&mut self) {
        let main = self.functions.get("main").unwrap();  // we presume there is a "main" function
        self.process_function(main);
    }

    fn process_function(&mut self, function: &Function) {
        if self.processed.contains(&function.name) {
            return;
        }
        self.processed.insert(function.name.clone());

        let semantics = self.summarize(function);  // summarize function
        println!("Function: {}; Semantics: {}", function.name, semantics);

        for call in &function.calls {
            let other = self.functions.get(call).unwrap();  // we presume function 'call' exists
            self.process_function(&other);
        }
    }

    fn summarize(&self, _function: &Function) -> String {
        // TODO: Implement function summarization. Now it just returns a stub
        "Function Summary".to_string()
    }
}