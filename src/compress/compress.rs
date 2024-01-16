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
    pub packages: HashMap<String, Package>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Package {
    #[serde(rename = "Functions")]
    pub functions: HashMap<String, Function>,
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

    // compress_data
    compress_data: Option<String>,
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

    // compress_data
    compress_data: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Identity {
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
}

pub fn from_json(json: &str) -> Result<Repository, Box<dyn Error>> {
    let f: Repository = serde_json::from_str(json)?;
    Ok(f)
}


pub async fn compress_all(repo: &mut Repository) {
    let mut to_compress = Vec::new();
    for (_, pkg) in &repo.packages {
        for (_, func) in &pkg.functions {
            let id = Identity { pkg_path: func.pkg_path.clone(), name: func.name.clone() };
            to_compress.push(id)
        }
    }

    for id in to_compress {
        cascade_compress_function(&id, repo).await;
    }
}

#[async_recursion]
pub async fn cascade_compress_function(id: &Identity, repo: &mut Repository) {
    let mut to_compress = Vec::new();

    {
        let func_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().functions.get(id.name.as_str());
        if func_opt.is_none() {
            println!("not found function, id {:?}", id);
        }
        let func_opt = func_opt.unwrap();
        if func_opt.compress_data.is_some() {
            println!("{} is already compressed, skip it.", func_opt.name);
            return;
        }
        if let Some(calls) = &func_opt.internal_function_calls {
            for (_, f) in calls {
                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
            }
        }

        if let Some(calls) = &func_opt.internal_method_calls {
            for (_, f) in calls {
                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
            }
        }
    }

    for f_id in to_compress {
        cascade_compress_function(&f_id, repo).await;
    }

    let mut map = HashMap::new();
    let content = {
        let func_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().functions.get(id.name.as_str()).unwrap();
        if let Some(calls) = &func_opt.internal_function_calls {
            for (k, f) in calls {
                let sub_function = repo.packages.get(f.pkg_path.as_str()).unwrap().functions.get(f.name.as_str()).unwrap();
                map.insert(k.clone(), sub_function.compress_data.clone().unwrap());
            }
        }

        if let Some(calls) = &func_opt.internal_method_calls {
            for (k, f) in calls {
                let sub_function = repo.packages.get(f.pkg_path.as_str()).unwrap().functions.get(f.name.as_str()).unwrap();
                map.insert(k.clone(), sub_function.compress_data.clone().unwrap());
            }
        }

        println!("start to compress function: {}", func_opt.name);
        llm_compress(func_opt.content.as_str(), map).await
    };

    let mut func_opt = repo.packages.get_mut(id.pkg_path.as_str()).unwrap().functions.get_mut(id.name.as_str()).unwrap();
    func_opt.compress_data = content;
}

async fn llm_compress(func: &str, extra: HashMap<String, String>) -> Option<String> {
    let compress_data = _ollama_compress(func.to_string(), extra).await;
    Option::from(compress_data)
}

pub async fn _ollama_compress(func: String, ctx: HashMap<String, String>) -> String {
    let request_url = format!("http://localhost:11434/api/generate");

    let mut prompt = r#"You are an engineer who is proficient in Golang. You are responsible for summarizing the functions/methods given by the user, DO NOT put anything that is not mentioned in the function.Try to condense output into one sentence and retain key information as much as possible. DO NOT show any codes in your answer. Function/methods content is as follow:"#.to_string();

    prompt.push_str("\n```\n");
    prompt.push_str(func.as_str());
    prompt.push_str("\n```\n");
    if !ctx.is_empty() {
        prompt.push_str("\nRelated function:\n");
        for (name, compressed_data) in ctx {
            prompt.push_str(&*(name + ": " + &*compressed_data + "\n"));
        }
    }


    println!("use prompt:\n{}", prompt);
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