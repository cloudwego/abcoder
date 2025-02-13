// Copyright 2025 CloudWeGo Authors
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     https://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use std::collections::HashMap;
use std::error::Error;
use std::ops::Add;

use async_recursion::async_recursion;
// Add these imports at the beginning of your file

use types::types::{
    CalledType, Identity, KeyValueType, Repository, ToCompressFunc, ToCompressType,
};

use crate::compress::llm::compress;
use crate::compress::llm::ToCompress;

use crate::compress::types;
use crate::config::CONFIG;
use crate::storage::cache::get_cache;
use crate::storage::cache::load_repo;

use super::types::types::ToCompressVar;
use super::types::types::Variant;

pub fn from_json(id: &str, json: &str) -> Result<Repository, Box<dyn Error>> {
    let mut f: Repository = serde_json::from_str(json)?;
    if f.id == "" {
        f.id = id.to_string();
    }
    if f.graph.is_none() {
        f.build_graph();
    }
    f.save_to_cache();
    Ok(f)
}

pub async fn compress_all(repo: &mut Repository) {
    let mut to_compress_func = Vec::new();
    let mut to_compress_type = Vec::new();
    let mut to_compress_var = Vec::new();

    for (_, _mod) in &repo.modules {
        if _mod.dir == "" {
            // NOTICE: empty dir means it's a external module, which is only used for lookup symbols
            continue;
        }
        for (_, pkg) in &_mod.packages {
            for (_, func) in &pkg.functions {
                let id = func.id();
                to_compress_func.push(id)
            }

            for (_, _type) in &pkg.types {
                let id = _type.id();
                to_compress_type.push(id)
            }

            for (_, var) in &pkg.vars {
                let id = var.id();
                to_compress_var.push(id)
            }
        }
    }

    for id in to_compress_var {
        let mut m: HashMap<String, bool> = HashMap::new();
        cascade_compress_variable(&id, repo, &mut m).await;
    }

    for id in to_compress_func {
        let mut m: HashMap<String, bool> = HashMap::new();
        cascade_compress_function(&id, repo, &mut m).await;
    }

    for id in to_compress_type {
        let mut m: HashMap<String, bool> = HashMap::new();
        cascade_compress_struct(&id, repo, &mut m).await;
    }

    for (mname, _mod) in &repo.clone().modules {
        if _mod.dir == "" {
            // NOTICE: empty dir means it's a external module, which is only used for lookup symbols
            continue;
        }
        for (id, pkg) in &_mod.packages {
            if pkg.compress_data.is_none() {
                compress_package(&id, mname, repo).await;
            }
        }
    }
}

pub async fn compress_package(id: &str, module: &str, repo: &mut Repository) {
    let pkg = repo
        .modules
        .get_mut(module)
        .unwrap()
        .packages
        .get_mut(id)
        .unwrap();
    let compress_data = pkg.to_compress();
    let compress_data =
        llm_compress_package(serde_json::to_string(&compress_data).unwrap().as_str()).await;
    if compress_data.is_none() {
        return;
    }
    let compress_data = compress_data.unwrap();
    pkg.compress_data = Some(compress_data);
    repo.save_to_cache();
    println!("finish to compress package: {}", id);
}

pub fn should_compress(id: &Identity, repo: &Repository) -> bool {
    if !id.inside() || id.pkg_path.contains("kitex_gen/") || id.pkg_path.contains("hertz_gen/") {
        return false;
    } else {
        let fi = repo.get_file_line(id);
        for exclude in &CONFIG.exclude_dirs {
            // check prefix
            if fi.file.starts_with(exclude) {
                return false;
            }
        }
        return true;
    }
}

const MAX_REFERS: usize = 4;

pub async fn cascade_compress_variable(
    id: &Identity,
    repo: &mut Repository,
    m: &mut HashMap<String, bool>,
) {
    if !should_compress(id, &repo) {
        return;
    }
    println!("start to comprees:{:?}", id);
    let mut refs = Vec::new();
    let var_opt = {
        let v = repo.get_var(id);
        if v.is_none() {
            eprintln!("not found var, id {:?}", id);
            return;
        }
        v.unwrap().clone()
    };

    if let Some(d) = &var_opt.compress_data {
        if d != "" {
            return;
        }
    }

    let graph = repo.graph.as_ref().unwrap();
    let var_node = graph.get(&String::from(id));
    if var_node.is_none() {
        eprintln!("var node not found, id {:?}", id);
        return;
    }
    let var_node = var_node.unwrap();
    for (i, v) in var_node.references.iter().enumerate() {
        if i >= MAX_REFERS {
            eprintln!("too many references for {:?}", id);
            break;
        }
        let c = repo.get_id_content(v);
        if c.is_none() {
            eprintln!("{:?} node is not found", v);
            continue;
        }
        let elem = c.unwrap();
        refs.push(elem);
    }

    // compress type if any
    let ty: Option<String> = if let Some(id) = &var_opt.type_id {
        cascade_compress_struct(id, repo, m).await;
        let tt = repo.get_type(id).clone();
        if tt.is_none() {
            eprintln!("not found ty {:?}", id);
            None
        } else {
            tt.unwrap().compress_data.clone()
        }
    } else {
        None
    };

    if let Some(c) = llm_compress_var(&var_opt.content, refs, &ty).await {
        if c != "" {
            let var = repo
                .modules
                .get_mut(&id.mod_path)
                .unwrap()
                .packages
                .get_mut(id.pkg_path.as_str())
                .unwrap()
                .vars
                .get_mut(id.name.as_str())
                .unwrap();
            let content = c.trim().to_string();
            var.compress_data = Some(content);
            repo.save_to_cache();
            return;
        }
    }
    panic!("empty compress for {:?}", id)
}

#[async_recursion]
pub async fn cascade_compress_function(
    id: &Identity,
    repo: &mut Repository,
    m: &mut HashMap<String, bool>,
) {
    if !should_compress(id, repo) {
        return;
    }
    println!("start to comprees:{:?}", id);
    let mut to_compress_func = Vec::new();
    let mut to_compress_type = Vec::new();
    let mut to_compress_var = Vec::new();

    {
        let func_opt = repo.get_func(id);
        if func_opt.is_none() {
            eprintln!("not found function, id {:?}", id);
            return;
        }
        let func_opt = func_opt.unwrap();

        // Already compress path
        if let Some(d) = &func_opt.compress_data {
            if d != "" {
                return;
            }
        }

        // vars
        if let Some(vars) = &func_opt.global_vars {
            for (f) in vars {
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_var.push(id);
            }
        }

        // Start to compress internal function callls
        if let Some(calls) = &func_opt.function_calls {
            for f in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive function: {}", f.name);
                    continue;
                }
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_func.push(id);
            }
        }

        // Start to compress internal method_calls
        if let Some(calls) = &func_opt.method_calls {
            for f in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive method: {}", f.name);
                    continue;
                }
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_func.push(id);
            }
        }

        // params
        if let Some(calls) = &func_opt.params {
            for f in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive function: {}", f.name);
                    continue;
                }
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_type.push(id);
            }
        }

        // rets
        if let Some(calls) = &func_opt.results {
            for f in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive function: {}", f.name);
                    continue;
                }
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_type.push(id);
            }
        }

        // types
        if let Some(calls) = &func_opt.types {
            for f in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive function: {}", f.name);
                    continue;
                }
                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a calling cycle: {}", compress_key);
                    continue;
                }
                let id = f.clone();
                m.insert(compress_key, true);
                to_compress_type.push(id);
            }
        }

        // receiver
        if let Some(f) = &func_opt.receiver {
            let compress_key = String::from(&f.type_id);
            if !m.get(&compress_key).is_some() {
                let id = f.type_id.clone();
                m.insert(compress_key, true);
                to_compress_type.push(id);
            }
        }
    }

    // Recursive call
    for f_id in to_compress_var {
        cascade_compress_variable(&f_id, repo, m).await;
        m.remove(&f_id.to_string());
    }

    for f_id in to_compress_func {
        cascade_compress_function(&f_id, repo, m).await;
        m.remove(&f_id.to_string());
    }

    for t_id in to_compress_type {
        cascade_compress_struct(&t_id, repo, m).await;
        m.remove(&t_id.to_string());
    }

    let mut func_map = HashMap::new();
    let mut type_map = HashMap::new();
    let mut var_map = HashMap::new();
    let mut inputs_map = HashMap::new();
    let mut outputs_map = HashMap::new();
    let mut receiver: Option<String> = None;

    let content = {
        let func_opt = repo.get_func(id).unwrap();

        // func body vars
        if let Some(vars) = &func_opt.global_vars {
            for (f) in vars {
                let var = repo.get_var(f);
                if var.is_none() {
                    eprintln!("not found var, id {:?}", id);
                    continue;
                }
                let var = var.unwrap();
                if var.compress_data.is_none() || var.compress_data.as_ref().unwrap() == "" {
                    eprintln!("var {}.{} is not compressed!!!", var.pkg_path, var.name);
                    var_map.insert(f.name.clone(), var.content.clone());
                } else {
                    var_map.insert(f.name.clone(), var.compress_data.clone().unwrap());
                }
            }
        }

        // func body types
        if let Some(types) = &func_opt.types {
            for (f) in types {
                let sub_type = repo.get_type(f);
                if sub_type.is_none() {
                    eprintln!("not found type, id {:?}", id);
                    continue;
                }
                let sub_type = sub_type.unwrap();
                if sub_type.compress_data.is_none()
                    || sub_type.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub type {}.{} is not compressed!!!",
                        sub_type.pkg_path, sub_type.name
                    );
                    type_map.insert(sub_type.name.clone(), sub_type.content.clone());
                } else {
                    type_map.insert(
                        sub_type.name.clone(),
                        sub_type.compress_data.clone().unwrap(),
                    );
                }
            }
        }

        // params
        if let Some(types) = &func_opt.params {
            for (f) in types {
                let sub_type = repo.get_type(f);
                if sub_type.is_none() {
                    eprintln!("not found type, id {:?}", id);
                    continue;
                }
                let sub_type = sub_type.unwrap();
                if sub_type.compress_data.is_none()
                    || sub_type.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub type {}.{} is not compressed!!!",
                        sub_type.pkg_path, sub_type.name
                    );
                    inputs_map.insert(sub_type.name.clone(), sub_type.content.clone());
                } else {
                    inputs_map.insert(
                        sub_type.name.clone(),
                        sub_type.compress_data.clone().unwrap(),
                    );
                }
            }
        }

        // results
        if let Some(types) = &func_opt.results {
            for (f) in types {
                let sub_type = repo.get_type(f);
                if sub_type.is_none() {
                    eprintln!("not found type, id {:?}", id);
                    continue;
                }
                let sub_type = sub_type.unwrap();
                if sub_type.compress_data.is_none()
                    || sub_type.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub type {}.{} is not compressed!!!",
                        sub_type.pkg_path, sub_type.name
                    );
                    outputs_map.insert(sub_type.name.clone(), sub_type.content.clone());
                } else {
                    outputs_map.insert(
                        sub_type.name.clone(),
                        sub_type.compress_data.clone().unwrap(),
                    );
                }
            }
        }

        // receiver
        if let Some(f) = &func_opt.receiver {
            let sub_type = repo.get_type(&f.type_id);
            if sub_type.is_none() {
                eprintln!("not found type, id {:?}", f.type_id);
            } else {
                let sub_type = sub_type.unwrap();
                if sub_type.compress_data.is_none()
                    || sub_type.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub type {}.{} is not compressed!!!",
                        sub_type.pkg_path, sub_type.name
                    );
                    receiver = Some(sub_type.content.clone());
                } else {
                    receiver = Some(sub_type.compress_data.clone().unwrap());
                }
            }
        }

        // Add the compress data of internal function calls
        if let Some(calls) = &func_opt.function_calls {
            for (f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive function: {}", f.name);
                    continue;
                }

                let sub_function = repo.get_func(f);
                if sub_function.is_none() {
                    eprintln!("not found function, id {:?}", id);
                    continue;
                }
                let sub_function = sub_function.unwrap();
                if sub_function.compress_data.is_none()
                    || sub_function.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub function {}.{} is not compressed!!!",
                        sub_function.pkg_path, sub_function.name
                    );
                    func_map.insert(sub_function.name.clone(), sub_function.content.clone());
                } else {
                    func_map.insert(
                        sub_function.name.clone(),
                        sub_function.compress_data.clone().unwrap(),
                    );
                }
            }
        }

        // Add the compress data of internal method calls
        if let Some(calls) = &func_opt.method_calls {
            for (f) in calls {
                if f.name == id.name && f.pkg_path == id.pkg_path {
                    eprintln!("find a recursive method: {}", f.name);
                    continue;
                }

                let sub_function = repo.get_func(id);
                if sub_function.is_none() {
                    eprintln!("not found function, id {:?}", id);
                    continue;
                }
                let sub_function = sub_function.unwrap();
                if sub_function.compress_data.is_none()
                    || sub_function.compress_data.as_ref().unwrap() == ""
                {
                    eprintln!(
                        "sub function {}.{} is not compressed!!!",
                        sub_function.pkg_path, sub_function.name
                    );
                    func_map.insert(sub_function.name.clone(), sub_function.content.clone());
                } else {
                    func_map.insert(
                        sub_function.name.clone(),
                        sub_function.compress_data.clone().unwrap(),
                    );
                }
            }
        }

        if func_opt.content.is_empty() {
            eprintln!("content is empty skip it");
            Some("".to_string())
        } else {
            llm_compress_func(
                func_opt.content.as_str(),
                func_map,
                type_map,
                var_map,
                inputs_map,
                outputs_map,
                receiver,
            )
            .await
        }
    };

    let func_opt = repo
        .modules
        .get_mut(&id.mod_path)
        .unwrap()
        .packages
        .get_mut(id.pkg_path.as_str())
        .unwrap()
        .functions
        .get_mut(id.name.as_str())
        .unwrap();
    if let Some(c) = content {
        if c != "" {
            let content = c.trim().to_string();
            func_opt.compress_data = Some(content);
            repo.save_to_cache();
            return;
        }
    }
    panic!("empty compress for {:?}", id)
}

#[async_recursion]
pub async fn cascade_compress_struct(
    id: &Identity,
    repo: &mut Repository,
    m: &mut HashMap<String, bool>,
) {
    if !should_compress(id, &repo) {
        return;
    }
    println!("start to comprees:{:?}", id);
    let mut to_compress = Vec::new();
    let mut to_compress_func = Vec::new();
    {
        let md = repo.modules.get(&id.mod_path);
        if md.is_none() {
            eprintln!("not found module, id {:?}", id);
            return;
        }
        let p = md.unwrap().packages.get(id.pkg_path.as_str());
        if p.is_none() {
            eprintln!("not found package, id {:?}", id);
            return;
        }
        let struct_opt = p.unwrap().types.get(id.name.as_str());
        if struct_opt.is_none() {
            eprintln!("not found struct, id {:?}", id);
            return;
        }
        let stru = struct_opt.unwrap();

        // Already compress path
        if let Some(d) = &stru.compress_data {
            if d != "" {
                return;
            }
        }

        // Start to compress sub struct
        if let Some(sub) = &stru.sub_struct {
            for f in sub {
                // TODO: compress extrenal symbol too
                if !repo.contains(f) {
                    continue;
                }

                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a struct embeding cycle: {}", compress_key);
                    continue;
                }

                let id = f.clone();
                to_compress.push(id);
                m.insert(compress_key, true);
            }
        }

        // Start to compress inline struct
        if let Some(inline) = &stru.inline_struct {
            for f in inline {
                // TODO: compress extrenal symbol too
                if !repo.contains(f) {
                    continue;
                }

                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a struct embeding cycle: {}", compress_key);
                    continue;
                }

                let id = f.clone();
                to_compress.push(id);
                m.insert(compress_key, true);
            }
        }

        // Start to compress related methods
        if let Some(methods) = &stru.methods {
            for (_, f) in methods {
                if !repo.contains(f) {
                    continue;
                }

                let compress_key = String::from(f);
                if m.get(&compress_key).is_some() {
                    eprintln!("find a struct embeding cycle: {}", compress_key);
                    continue;
                }

                let id = f.clone();
                to_compress_func.push(id);
                m.insert(compress_key, true);
            }
        }
    }

    // Recursive call
    for f_id in to_compress {
        cascade_compress_struct(&f_id, repo, m).await;
        m.remove(&f_id.to_string());
    }
    for f_id in to_compress_func {
        cascade_compress_function(&f_id, repo, m).await;
        m.remove(&f_id.to_string());
    }

    // Recursive compressing has done, start to compress myself.
    let mut type_map = HashMap::new();
    let mut method_map = HashMap::new();
    let content = {
        let _type = repo.get_type(id);
        if _type.is_none() {
            eprintln!("not found type, id {:?}", id);
            return;
        }
        let _type = _type.unwrap();

        // Add the compress data of sub struct
        if let Some(subs) = &_type.sub_struct {
            for (f) in subs {
                let sub = repo.get_type(f);
                if sub.is_none() {
                    eprintln!("do not get tye type in the pkg: {:?}", f);
                    continue;
                }
                let compress_data = sub.unwrap().compress_data.clone();
                if compress_data.is_none() || compress_data.as_ref().unwrap() == "" {
                    type_map.insert(f.name.clone(), sub.unwrap().content.clone());
                } else {
                    type_map.insert(f.name.clone(), compress_data.unwrap());
                }
            }
        }

        // Add the compress data of inline struct
        if let Some(inlines) = &_type.inline_struct {
            for (f) in inlines {
                let inline = repo.get_type(f);
                if inline.is_none() {
                    eprintln!("do not get tye type in the pkg: {:?}", f);
                    continue;
                }
                let compress_data = inline.unwrap().compress_data.clone();
                if compress_data.is_none() || compress_data.as_ref().unwrap() == "" {
                    eprint!("do not get the compress data of the inline struct: {:?}", f);
                    type_map.insert(f.name.clone(), inline.unwrap().content.clone());
                } else {
                    type_map.insert(f.name.clone(), compress_data.unwrap());
                }
            }
        }

        // Add the compress data of related methods(We have done function and methods compress before, so don't worry about it.)
        if let Some(methods) = &_type.methods {
            for (k, f) in methods {
                if !repo.contains(f) {
                    continue;
                }
                let func = repo.get_func(f);
                if func.is_none() || func.as_ref().unwrap().compress_data.is_none() {
                    eprintln!("do not get the method of the type, id: {:?}", f);
                    method_map.insert(k.clone(), func.unwrap().content.clone());
                } else {
                    method_map.insert(k.clone(), func.unwrap().compress_data.clone().unwrap());
                }
            }
        }

        if _type.content.is_empty() {
            eprintln!("content is empty skip it");
            Some("".to_string())
        } else {
            llm_compress_type(_type.content.as_str(), type_map, method_map).await
        }
    };

    let mut type_opt = repo
        .modules
        .get_mut(&id.mod_path)
        .unwrap()
        .packages
        .get_mut(id.pkg_path.as_str())
        .unwrap()
        .types
        .get_mut(id.name.as_str())
        .unwrap();
    if let Some(c) = content {
        if c != "" {
            let content = c.trim().to_string();
            type_opt.compress_data = Some(content);
            repo.save_to_cache();
            return;
        }
    }
    // panic!("empty compress for {:?}", id)
}

async fn llm_compress_package(pkg: &str) -> Option<String> {
    let compress_pkg = ToCompress::ToCompressPkg(pkg.to_string());
    let compress_data = compress(&compress_pkg).await;
    Option::from(compress_data)
}

async fn llm_compress_var(var: &String, refs: Vec<String>, ty: &Option<String>) -> Option<String> {
    let compress_var = ToCompressVar {
        content: var,
        r#type: ty,
        refers: refs,
    };
    let to_compress_str = serde_json::to_string(&compress_var).unwrap();
    let compress_enum = ToCompress::ToCompressVar(to_compress_str);
    let compress_data = compress(&compress_enum).await;
    Some(compress_data)
}

async fn llm_compress_func(
    func: &str,
    funcs: HashMap<String, String>,
    types: HashMap<String, String>,
    vars: HashMap<String, String>,
    inputs: HashMap<String, String>,
    outputs: HashMap<String, String>,
    receiver: Option<String>,
) -> Option<String> {
    let mut compress_func = ToCompressFunc {
        content: func.to_string(),
        related_func: None,
        related_type: None,
        related_var: None,
        receiver: None,
        params: None,
        results: None,
    };
    if !funcs.is_empty() {
        let mut related_func = Vec::new();
        for (name, compressed_data) in funcs {
            let re = CalledType {
                call_name: name,
                description: compressed_data,
            };
            related_func.push(re);
        }
        compress_func.related_func = Some(related_func);
    }
    if !types.is_empty() {
        let mut related_type = Vec::new();
        for (name, compressed_data) in types {
            let re = KeyValueType {
                name: name,
                description: compressed_data,
            };
            related_type.push(re);
        }
        compress_func.related_type = Some(related_type);
    }
    if !vars.is_empty() {
        let mut related_var = Vec::new();
        for (name, compressed_data) in vars {
            let re = KeyValueType {
                name: name,
                description: compressed_data,
            };
            related_var.push(re);
        }
        compress_func.related_var = Some(related_var);
    }
    if !inputs.is_empty() {
        let mut params = Vec::new();
        for (name, compressed_data) in inputs {
            let re = KeyValueType {
                name: name,
                description: compressed_data,
            };
            params.push(re);
        }
        compress_func.params = Some(params);
    }
    if !outputs.is_empty() {
        let mut results = Vec::new();
        for (name, compressed_data) in outputs {
            let re = KeyValueType {
                name: name,
                description: compressed_data,
            };
            results.push(re);
        }
        compress_func.results = Some(results);
    }
    if let Some(r) = receiver {
        compress_func.receiver = Some(r);
    }
    let to_compress_str = serde_json::to_string(&compress_func).unwrap();
    let compress_func_enum = ToCompress::ToCompressFunc(to_compress_str);
    let compress_data = compress(&compress_func_enum).await;
    Option::from(compress_data)
}

// depends on the compressed info of methods, so call llm_compress_func first.
async fn llm_compress_type(
    func: &str,
    extra_type: HashMap<String, String>,
    related_methods: HashMap<String, String>,
) -> Option<String> {
    let mut compress_type = ToCompressType {
        content: func.to_string(),
        related_methods: None,
        related_types: None,
    };
    if !extra_type.is_empty() {
        let mut r_type = Vec::new();
        for (name, compressed_data) in extra_type {
            let re = KeyValueType {
                name,
                description: compressed_data,
            };
            r_type.push(re);
        }
        compress_type.related_types = Some(r_type);
    }

    if !related_methods.is_empty() {
        let mut r_methods = Vec::new();
        for (name, compressed_data) in related_methods {
            let re = KeyValueType {
                name,
                description: compressed_data,
            };
            r_methods.push(re);
        }
        compress_type.related_methods = Some(r_methods);
    }

    let to_compress_str = serde_json::to_string(&compress_type).unwrap();
    let compress_type_enum = ToCompress::ToCompressType(to_compress_str);
    let compress_data = compress(&compress_type_enum).await;
    Option::from(compress_data)
}

fn pkg_name_to_repo_name(s: &str) -> String {
    let parts: Vec<_> = s.split('/').collect();

    match parts.len() {
        0 | 1 | 2 => String::from(s),
        _ => parts[1..3].join("/"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pkg_name_to_repo_name() {
        assert_eq!(pkg_name_to_repo_name("a/b/c"), "b/c");
        assert_eq!(pkg_name_to_repo_name("a/b/c/d"), "b/c");
        assert_eq!(pkg_name_to_repo_name("rust"), "rust");
    }
}
