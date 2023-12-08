use std::collections::HashMap;

struct Function {
    name: String,
    content: String,
    calls: Vec<String>,  // names of the functions this one calls
}

struct Project {
    functions: HashMap<String, Function>,
}

impl Project {
    pub fn new() -> Self {
        Self { functions: HashMap::new() }
    }

    pub fn add_function(&mut self, name: String, content: String, calls: Vec<String>) {
        let function = Function { name: name.clone(), content, calls };
        self.functions.insert(name, function);
    }

    pub fn process_functions(&self) {
        let main = self.functions.get("main").unwrap();  // we presume there is a "main" function
        self.process_function(main);
    }

    fn process_function(&self, function: &Function) {
        let semantics = self.summarize(function);  // summarize function
        println!("Function: {}; Semantics: {}", function.name, semantics);

        for call in &function.calls {
            let other = self.functions.get(call).unwrap();  // we presume function 'call' exists
            self.process_function(&other);
        }
    }

    fn summarize(&self, _function: &Function) -> String {
        // TODO: Implement function summarization. Now it just returns a stub
        "function summary".to_string()
    }
}