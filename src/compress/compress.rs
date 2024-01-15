use std::collections::HashMap;
use std::error::Error;
use std::ops::Add;

use async_recursion::async_recursion;
// Add these imports at the beginning of your file
use serde::{Deserialize, Serialize};

use crate::compress::compress;

#[derive(Serialize, Deserialize, Debug)]
pub struct Repository {
    #[serde(rename = "ModName")]
    mod_name: String,
    #[serde(rename = "Packages")]
    packages: HashMap<String, Package>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Package {
    #[serde(rename = "Functions")]
    functions: HashMap<String, Function>,
    #[serde(rename = "Types")]
    types: HashMap<String, Struct>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Function {
    #[serde(rename = "IsMethod")]
    is_method: bool,
    #[serde(rename = "PkgPath")]
    pkg_path: String,
    #[serde(rename = "Name")]
    name: String,
    #[serde(rename = "Content")]
    content: String,
    #[serde(rename = "AssociatedStruct")]
    associated_struct: Option<Identity>,
    #[serde(rename = "InternalFunctionCalls")]
    internal_function_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "ThirdPartyFunctionCalls")]
    third_party_function_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "InternalMethodCalls")]
    internal_method_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "ThirdPartyMethodCalls")]
    third_party_method_calls: Option<HashMap<String, Identity>>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Struct {
    #[serde(rename = "TypeKind")]
    type_kind: u8,
    #[serde(rename = "PkgPath")]
    pkg_path: String,
    #[serde(rename = "Name")]
    name: String,
    #[serde(rename = "Content")]
    content: String,
    #[serde(rename = "SubStruct")]
    sub_struct: Option<HashMap<String, Identity>>,
    #[serde(rename = "InlineStruct")]
    inline_struct: Option<HashMap<String, Identity>>,
    #[serde(rename = "Methods")]
    methods: Option<HashMap<String, Identity>>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Identity {
    #[serde(rename = "PkgPath")]
    pkg_path: String,
    #[serde(rename = "Name")]
    name: String,
}

pub fn from_json(json: &str) -> Result<Repository, Box<dyn Error>> {
    let f: Repository = serde_json::from_str(json)?;
    Ok(f)
}


#[async_recursion]
pub async fn cascade_compress_function(func: &mut Function) {
    // if func.internal_function_calls.is_none() {
    //     llm_compress(func);
    //     return;
    // }
    //
    // for (_, mut f) in func.internal_function_calls.as_mut().unwrap() {
    //     if f.compress_info.is_none() {
    //         cascade_compress_function(&mut f).await
    //     }
    // }
    //
    // llm_compress(func).await;
    //
    // return;
}

async fn llm_compress(func: &mut Function) {
    // let mut map = HashMap::new(); // 创建一个空的 HashMap
    // if func.internal_function_calls.is_some() {
    //     for (k, ff) in func.internal_function_calls.as_ref().unwrap() {
    //         map.insert(k.clone(), ff.compress_info.clone().unwrap());
    //     }
    // }
    //
    // let compress_data = _ollama_compress(func.content.clone(), map).await;
    // func.compress_info = Option::from(compress_data);
}

pub async fn _ollama_compress(func: String, ctx: HashMap<String, String>) -> String {
    let request_url = format!("http://localhost:11434/api/generate");

    let mut prompt = r#"You are an engineer who is proficient in Golang. You are responsible for summarizing the functions/methods given by the user.Try to condense output into one sentence and retain key information as much as possible. DO NOT show any codes in your answer. Function/methods content is as follow:"#.to_string();

    prompt.push_str("\n");
    prompt.push_str(func.as_str());
    prompt.push_str("\nRelated function:\n");
    for (name, compressed_data) in ctx {
        prompt.push_str(&*(name + ": " + &*compressed_data + "\n"));
    }


    let req_body: ollama_req = ollama_req { model: "codellama".to_string(), prompt };
    let client = reqwest::Client::new();
    let mut response = client
        .post(&request_url)
        .json(&req_body)
        .send()
        .await.unwrap();


    let mut output = String::new();
    while let Ok(Some(chunk)) = response.chunk().await {
        let result = serde_json::from_slice(&chunk);
        if result.is_err() {
            break;
        }

        let value: ollama_resp = result.unwrap();

        if !value.response.is_empty() {
            output.push_str(value.response.as_str());
        }

        if value.done {
            break;
        }
    }

    output
}

#[derive(Serialize, Deserialize, Debug)]
struct ollama_req {
    model: String,
    prompt: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct ollama_resp {
    model: String,
    created_at: String,
    response: String,
    done: bool,
}