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

#[derive(Serialize, Deserialize, Debug)]
struct ToCompressFunc {
    #[serde(rename = "Content")]
    content: String,
    #[serde(rename = "Related_func")]
    related_func: Option<Vec<CalledType>>,
}

#[derive(Serialize, Deserialize, Debug)]
struct ToCompressType {
    #[serde(rename = "Content")]
    content: String,
    #[serde(rename = "Related_methods")]
    related_methods: Option<Vec<KeyValueType>>,
    #[serde(rename = "Related_types")]
    related_types: Option<Vec<KeyValueType>>,
}


#[derive(Serialize, Deserialize, Debug)]
struct CalledType {
    #[serde(rename = "CallName")]
    pub call_name: String,
    #[serde(rename = "Description")]
    pub description: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct KeyValueType {
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Description")]
    pub description: String,
}

pub fn from_json(json: &str) -> Result<Repository, Box<dyn Error>> {
    let f: Repository = serde_json::from_str(json)?;
    Ok(f)
}


pub async fn compress_all(repo: &mut Repository) {
    let mut to_compress_func = Vec::new();
    let mut to_compress_type = Vec::new();

    for (_, pkg) in &repo.packages {
        for (_, func) in &pkg.functions {
            let id = Identity { pkg_path: func.pkg_path.clone(), name: func.name.clone() };
            to_compress_func.push(id)
        }

        for (_, _type) in &pkg.types {
            let id = Identity { pkg_path: _type.pkg_path.clone(), name: _type.name.clone() };
            to_compress_type.push(id)
        }
    }

    for id in to_compress_func {
        cascade_compress_function(&id, repo).await;
    }

    for id in to_compress_type {
        cascade_compress_struct(&id, repo).await;
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
        if func_opt.content.is_empty() {
            println!("content is empty skip it");
            Some("".to_string())
        } else {
            llm_compress_func(func_opt.content.as_str(), map).await
        }
    };

    let mut func_opt = repo.packages.get_mut(id.pkg_path.as_str()).unwrap().functions.get_mut(id.name.as_str()).unwrap();
    if content.is_some() {
        let content = content.unwrap().trim().to_string();
        func_opt.compress_data = Some(content);
        return;
    }
    func_opt.compress_data = content;
}

#[async_recursion]
pub async fn cascade_compress_struct(id: &Identity, repo: &mut Repository) {
    let mut to_compress = Vec::new();

    {
        let struct_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().types.get(id.name.as_str());
        if struct_opt.is_none() {
            println!("not found struct, id {:?}", id);
        }
        let stru = struct_opt.unwrap();
        if stru.compress_data.is_some() {
            println!("{} is already compressed, skip it.", stru.name);
            return;
        }
        if let Some(sub) = &stru.sub_struct {
            for (_, f) in sub {
                if !f.pkg_path.starts_with(&repo.mod_name) {
                    continue;
                }

                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
            }
        }

        if let Some(inline) = &stru.inline_struct {
            for (_, f) in inline {
                if !f.pkg_path.starts_with(&repo.mod_name) {
                    continue;
                }

                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
            }
        }
    }

    for f_id in to_compress {
        cascade_compress_struct(&f_id, repo).await;
    }

    let mut type_map = HashMap::new();
    let mut method_map = HashMap::new();
    let content = {
        let _type = repo.packages.get(id.pkg_path.as_str()).unwrap().types.get(id.name.as_str()).unwrap();
        if let Some(subs) = &_type.sub_struct {
            for (k, f) in subs {
                let pkg = repo.packages.get(f.pkg_path.as_str());
                if pkg.is_none() {
                    // TODO
                    eprintln!("do not get the type, must be a third party one: {:?}", f);
                    continue;
                }
                let sub = pkg.unwrap().types.get(f.name.as_str());
                type_map.insert(k.clone(), sub.unwrap().compress_data.clone().unwrap());
            }
        }

        if let Some(inlines) = &_type.inline_struct {
            for (k, f) in inlines {
                let pkg = repo.packages.get(f.pkg_path.as_str());
                if pkg.is_none() {
                    // TODO
                    eprintln!("do not get the type, must be a third party one: {:?}", f);
                    continue;
                }
                let inline = repo.packages.get(f.pkg_path.as_str()).unwrap().types.get(f.name.as_str());

                type_map.insert(k.clone(), inline.unwrap().compress_data.clone().unwrap());
            }
        }

        if let Some(methods) = &_type.methods {
            for (k, f) in methods {
                let func = repo.packages.get(f.pkg_path.as_str()).unwrap().functions.get(f.name.as_str());
                if func.is_none() {
                    // TODO
                    eprintln!("[BUG] do not get the method of the type, id: {:?}", f);
                } else {
                    method_map.insert(k.clone(), func.unwrap().compress_data.clone().unwrap());
                }
            }
        }

        println!("start to compress type: {}", _type.name);
        if _type.content.is_empty() {
            println!("content is empty skip it");
            Some("".to_string())
        } else {
            llm_compress_type(_type.content.as_str(), type_map, method_map).await
        }
    };

    let mut type_opt = repo.packages.get_mut(id.pkg_path.as_str()).unwrap().types.get_mut(id.name.as_str()).unwrap();
    if content.is_some() {
        let content = content.unwrap().trim().to_string();
        type_opt.compress_data = Some(content);
        return;
    }
    type_opt.compress_data = content;
}


pub enum ToCompress {
    ToCompressFunc(String),
    ToCompressType(String),
}

async fn llm_compress_func(func: &str, extra: HashMap<String, String>) -> Option<String> {
    let mut compress_func = ToCompressFunc { content: func.to_string(), related_func: None };
    if !extra.is_empty() {
        let mut related_func = Vec::new();
        for (name, compressed_data) in extra {
            let re = CalledType { call_name: name, description: compressed_data };
            related_func.push(re);
        }
        compress_func.related_func = Some(related_func);
    }
    let to_compress_str = serde_json::to_string(&compress_func).unwrap();
    let compress_func_enum = ToCompress::ToCompressFunc(to_compress_str);
    let compress_data = _ollama_compress(compress_func_enum).await;
    Option::from(compress_data)
}

// depends on the compressed info of methods, so call llm_compress_func first.
async fn llm_compress_type(func: &str, extra_type: HashMap<String, String>, related_methods: HashMap<String, String>) -> Option<String> {
    let mut compress_type = ToCompressType { content: func.to_string(), related_methods: None, related_types: None };
    if !extra_type.is_empty() {
        let mut r_type = Vec::new();
        for (name, compressed_data) in extra_type {
            let re = KeyValueType { name, description: compressed_data };
            r_type.push(re);
        }
        compress_type.related_types = Some(r_type);
    }

    if !related_methods.is_empty() {
        let mut r_methods = Vec::new();
        for (name, compressed_data) in related_methods {
            let re = KeyValueType { name, description: compressed_data };
            r_methods.push(re);
        }
        compress_type.related_methods = Some(r_methods);
    }


    let to_compress_str = serde_json::to_string(&compress_type).unwrap();
    let compress_type_enum = ToCompress::ToCompressType(to_compress_str);
    let compress_data = _ollama_compress(compress_type_enum).await;
    Option::from(compress_data)
}


pub async fn _ollama_compress(to_compress: ToCompress) -> String {
    let request_url = format!("http://localhost:11434/api/generate");
    let mut model_name = "codellama-private";
    let mut to_compress_str = String::new();
    match to_compress {
        ToCompress::ToCompressType(t) => {
            model_name = "codellama-private-type";
            to_compress_str = t;
        }
        ToCompress::ToCompressFunc(f) => {
            to_compress_str = f;
        }
    }

    println!("use prompt:\n{}", to_compress_str);
    let req_body: OllamaReq = OllamaReq { model: model_name.to_string(), prompt: to_compress_str };
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

        let value: OllamaResp = result.unwrap();

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
struct OllamaReq {
    model: String,
    prompt: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct OllamaResp {
    model: String,
    created_at: String,
    response: String,
    done: bool,
}