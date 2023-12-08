#[derive(Serialize, Deserialize, Debug)]
pub struct Function {
    pub(crate) name: String,
    pub(crate) is_method: bool,
    pub(crate) is_third_party: bool,
    pub(crate) associated_struct: String,
    pub(crate) content: String,
    pub(crate) compressed_content: CompressedFunction,
    pub(crate) calls: Vec<String>,  // names of the functions this one calls
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CompressedFunction {
    pub(crate) function_name: String,
    pub(crate) description: String,
    pub(crate) input: String,
    pub(crate) output: String,
    pub(crate) side_effect: String,
}

pub trait LanguageParser {
    fn new() -> Self;
    fn process(&mut self);
    fn summarize(&self, _function: &Function) -> String;
}