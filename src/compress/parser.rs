pub struct Function {
    pub(crate) name: String,
    pub(crate) content: String,
    pub(crate) calls: Vec<String>,  // names of the functions this one calls
}

pub trait LanguageParser {
    fn new() -> Self;
    fn add_function(&mut self, name: String, content: String, calls: Vec<String>);
    fn process_functions(&mut self);
    fn process_function(&mut self, function: &Function);
    fn summarize(&self, _function: &Function) -> String;
}