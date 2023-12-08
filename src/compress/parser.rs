pub struct Function {
    pub(crate) name: String,
    pub(crate) content: String,
    pub(crate) calls: Vec<String>,  // names of the functions this one calls
}

pub trait LanguageParser {
    fn new() -> Self;
    fn process(&mut self);
    fn summarize(&self, _function: &Function) -> String;
}