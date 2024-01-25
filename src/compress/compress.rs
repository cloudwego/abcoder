use std::collections::HashMap;
use std::error::Error;
use std::hash::Hash;
use std::ops::Add;

use async_recursion::async_recursion;
// Add these imports at the beginning of your file
use serde::{Deserialize, Serialize};

use llm::ollama::ollama_compress;
use types::types::{CalledType, Identity, KeyValueType, Repository, ToCompressFunc, ToCompressType};

use crate::compress::compress;
use crate::compress::llm;
use crate::compress::llm::coze::coze_compress;
use crate::compress::types;

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
        let mut m:HashMap<String, bool> = HashMap::new();
        cascade_compress_function(&id, repo, &mut m).await;
    }

    for id in to_compress_type {
        let mut m:HashMap<String, bool> = HashMap::new();
        cascade_compress_struct(&id, repo, &mut m).await;
    }
}

#[async_recursion]
pub async fn cascade_compress_function(id: &Identity, repo: &mut Repository, m: &mut HashMap<String,bool>) {
    let mut to_compress = Vec::new();

    {
        let func_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().functions.get(id.name.as_str());
        if func_opt.is_none() {
            println!("not found function, id {:?}", id);
            return;
        }
        let func_opt = func_opt.unwrap();
        if func_opt.compress_data.is_some() {
            println!("{} is already compressed, skip it.", func_opt.name);
            return;
        }
        if let Some(calls) = &func_opt.internal_function_calls {
            for (_, f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path{
                    println!("find a recursive function: {}",f.name);
                    continue;
                }
                let compress_key = f.pkg_path.clone().add(&f.name);
                if m.get(&compress_key).is_some(){
                    println!("find a calling cycle: {}",compress_key);
                    continue;
                }
                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                m.insert(compress_key,true);
                to_compress.push(id);
            }
        }

        if let Some(calls) = &func_opt.internal_method_calls {
            for (_, f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path{
                    println!("find a recursive method: {}",f.name);
                    continue;
                }
                let compress_key = f.pkg_path.clone().add(&f.name);
                if m.get(&compress_key).is_some(){
                    println!("find a calling cycle: {}",compress_key);
                    continue;
                }
                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                m.insert(compress_key,true);
                to_compress.push(id);
            }
        }
    }

    for f_id in to_compress {
        cascade_compress_function(&f_id, repo, m).await;
    }

    let mut map = HashMap::new();
    let content = {
        let func_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().functions.get(id.name.as_str()).unwrap();
        if let Some(calls) = &func_opt.internal_function_calls {
            for (k, f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path{
                    println!("find a recursive function: {}",f.name);
                    continue;
                }

                let sub_function = repo.packages.get(f.pkg_path.as_str()).unwrap().functions.get(f.name.as_str());
                if sub_function.is_none(){
                    println!("not found function, id {:?}", id);
                    continue;
                }
                let sub_function = sub_function.unwrap();
                map.insert(k.clone(), sub_function.compress_data.clone().unwrap());
            }
        }

        if let Some(calls) = &func_opt.internal_method_calls {
            for (k, f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path{
                    println!("find a recursive method: {}",f.name);
                    continue;
                }

                let sub_function = repo.packages.get(f.pkg_path.as_str()).unwrap().functions.get(f.name.as_str());
                if sub_function.is_none(){
                    println!("not found function, id {:?}", id);
                    continue;
                }
                let sub_function = sub_function.unwrap();
                let value = sub_function.compress_data.clone();
                if value.is_some(){
                    map.insert(k.clone(),value.unwrap());
                }
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
        repo.save_to_cache();
        return;
    }
    func_opt.compress_data = content;
    repo.save_to_cache();
}

#[async_recursion]
pub async fn cascade_compress_struct(id: &Identity, repo: &mut Repository, m: &mut HashMap<String,bool>) {
    let mut to_compress = Vec::new();

    {
        let struct_opt = repo.packages.get(id.pkg_path.as_str()).unwrap().types.get(id.name.as_str());
        if struct_opt.is_none() {
            println!("not found struct, id {:?}", id);
            return;
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

                let compress_key = f.pkg_path.clone().add(&f.name);
                if m.get(&compress_key).is_some(){
                    println!("find a struct embeding cycle: {}",compress_key);
                    continue;
                }

                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
                m.insert(compress_key, true);
            }
        }

        if let Some(inline) = &stru.inline_struct {
            for (_, f) in inline {
                if !f.pkg_path.starts_with(&repo.mod_name) {
                    continue;
                }

                let compress_key = f.pkg_path.clone().add(&f.name);
                if m.get(&compress_key).is_some(){
                    println!("find a struct embeding cycle: {}",compress_key);
                    continue;
                }

                let id = Identity { pkg_path: f.pkg_path.clone(), name: f.name.clone() };
                to_compress.push(id);
                m.insert(compress_key, true);
            }
        }
    }

    for f_id in to_compress {
        cascade_compress_struct(&f_id, repo, m).await;
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
                if sub.is_none(){
                    eprintln!("do not get tye type in the pkg: {:?}",f);
                    continue;
                }
                let compress_data = sub.unwrap().compress_data.clone();
                if compress_data.is_none(){
                    continue;
                }
                type_map.insert(k.clone(),compress_data.unwrap());
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
                if inline.is_none(){
                    eprintln!("do not get tye type in the pkg: {:?}",f);
                    continue;
                }
                let compress_data =  inline.unwrap().compress_data.clone();
                if compress_data.is_none(){
                    continue;
                }
                type_map.insert(k.clone(), compress_data.unwrap());
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
        repo.save_to_cache();
        return;
    }
    type_opt.compress_data = content;
    repo.save_to_cache();
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
    let compress_data = coze_compress(compress_func_enum).await;
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
    let compress_data = coze_compress(compress_type_enum).await;
    Option::from(compress_data)
}
