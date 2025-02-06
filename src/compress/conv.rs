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

use std::{
    borrow::BorrowMut,
    collections::{HashMap, HashSet},
    fs,
    path::{Path, PathBuf},
    sync::{Arc, Mutex},
};

use async_recursion::async_recursion;

use serde_json::to_string;
use syn::Item;

use crate::{
    compress::{
        rust::avoid_rust_keywords,
        types::types::{Reference, ToValidate},
    },
    utils::{
        self, cmd,
        files::{camel_to_snake, snake_to_camel},
    },
};

use crate::compress::llm::ToCompress;

use super::{
    llm::compress,
    rust::{
        convert_crate, extract_msg_from_err, new_rust_impt, normalize_rust_import,
        replace_impt_crate, Cargo,
    },
    types::types::{
        Code, CodeCache, FileLine, Function, Identity, Module, Node, NodeKind, Package, Repository,
        Struct, ToConvert, ToMerge, Variant,
    },
};

// TODO: meet more cases
fn go_impt_to_rust_impt(id: &Identity, repo: &Repository) -> String {
    if id.inside() {
        let (m, p) = repo.get_pkg(id).unwrap();
        let path = p.id.strip_prefix(&m.name).unwrap_or(&p.id);
        new_rust_impt("crate", path.strip_prefix("/").unwrap_or(path))
    } else {
        let m = id.mod_path.split("@").nth(0).unwrap_or(&id.mod_path);
        let _mod = convert_crate(m);
        let pkg = id.pkg_path.strip_prefix(m).unwrap_or(&id.pkg_path);
        let path = if pkg != "" {
            format!("{}{}", _mod, pkg)
        } else {
            _mod
        };
        // NOTICE: external id will be mocked inside crate::external_mocks::{{_mod}}::{{pkg}}
        new_rust_impt(&format!("crate/{}", DIR_EXTERN_MOCKS), &path)
    }
}

fn insert_to_mod_file(
    pkg_path: &str,
    content: &str,
    is_main: Option<bool>,
    pkg_files: Arc<Mutex<HashMap<String, (String, bool)>>>,
) {
    let mut file = {
        pkg_files
            .lock()
            .unwrap()
            .entry(String::from(pkg_path))
            .or_insert((String::new(), false))
            .clone()
    };
    file.0.push_str(content);
    if let Some(is_main) = is_main {
        file.1 = is_main;
    }
    // reset entry
    pkg_files
        .lock()
        .unwrap()
        .insert(String::from(pkg_path), file);
}

const MAX_LLM_FILE_SIZE: usize = 10 * 1024; // 10KB

const DIR_EXTERN_MOCKS: &str = "external_mocks";

pub async fn convert_go2rust(repo: &mut Repository, dir: &str, cache: &mut CodeCache) -> bool {
    // ensure dependency graph is built
    if repo.graph.is_none() {
        repo.build_graph();
    }

    // llm covert each id
    let graph = repo.graph.as_ref().unwrap();
    for (id, node) in graph {
        // cached codes
        if let Some(code) = cache.get(id) {
            // ensure code is cached
            if code.code != "" {
                continue;
            }
        }

        // NOTICE: external codes should be mocked in the crate::external_mocks
        // LLM convert internal one
        llm_go2rust_convert(node, repo, &mut cache.borrow_mut(), &mut HashSet::new()).await;
    }

    // collect crates
    let mut cargo = Cargo::new(&repo.id);
    for (_, code) in &cache.nodes {
        if let Some(deps) = &code.crates {
            cargo.dep(deps);
        }
    }

    // root dir
    let root_dir = Path::new(dir).join("src");
    let mut root_files = vec![root_dir.join("lib.rs")];
    let mut has_extern = false;

    // construct dirs and files
    for (mname, _mod) in &repo.modules {
        if mname == "" {
            continue;
        }

        // mod.rs (or main.rs) for each package
        let pkg_files = Arc::new(Mutex::new(HashMap::new()));

        // NOTICE: external mod is like 'mod_name@version'
        let external_mod = mname.contains("@");
        has_extern = has_extern || external_mod;

        // mod directory: {dir}/{mdir}/src
        let mdir = if _mod.dir == "." {
            root_dir.clone()
        } else if !external_mod {
            root_dir.join(Path::new(_mod.dir.as_str()))
        } else {
            // NOTICE: external mod will be mocked in crate::external_mocks
            root_dir
                .join(Path::new(DIR_EXTERN_MOCKS))
                .join(Path::new(&convert_crate(&_mod.name)))
        };

        for (_, pkg) in &_mod.packages {
            let pkg_name = pkg
                .id
                .split('/')
                .last()
                .unwrap_or(&pkg.id)
                .replace("-", "_");
            let pkg_path = pkg.id.strip_prefix(&_mod.name).unwrap_or(&pkg.id);
            let pkg_path = pkg_path.strip_prefix("/").unwrap_or(pkg_path);

            // create package dir
            let pdir = if pkg_path != "" {
                mdir.join(Path::new(pkg_path))
            } else {
                mdir.clone()
            };
            println!(
                "create dir {:?}, pkg {:?}, mod {:?}",
                pdir, pkg.id, _mod.name
            );
            fs::create_dir_all(&pdir).unwrap();

            // mod.rs (or main.rs or lib.rs)
            let mut pfile = (String::new(), pkg.is_main);

            // insert mod in parent's mod.rs
            if !pfile.1 {
                let parent = Path::new(pkg_path)
                    .parent()
                    .unwrap_or(Path::new(pkg_path))
                    .as_os_str()
                    .to_str()
                    .unwrap();
                insert_to_mod_file(
                    parent,
                    format!(
                        "{}mod {};\n",
                        if pkg.is_main { "" } else { "pub " },
                        avoid_rust_keywords(&pkg_name).unwrap_or(pkg_name)
                    )
                    .as_str(),
                    None,
                    pkg_files.clone(),
                );
            }

            let main_root = if let Some(main_pkg) = repo.inside_main_pkg(&mname, &pkg.id) {
                Some(go_impt_to_rust_impt(
                    &Identity {
                        mod_path: mname.clone(),
                        pkg_path: main_pkg.clone(),
                        name: "".to_string(),
                    },
                    repo,
                ))
            } else {
                None
            };

            let files = pkg.to_files();
            for (f, ids) in &files {
                let file = f.strip_suffix(".go").unwrap_or(f);
                let fkey = format!("{}/{}", pkg.id, file);

                // check file cache first
                let merged = if let Some(cs) = cache.files.get(&fkey) {
                    cs.clone()
                } else if ids.len() > 0 {
                    // merge Rust ASTs
                    let mut fcodes = Vec::new();
                    for id in ids {
                        if let Some(code) = cache.get_by_id(&id) {
                            fcodes.push((repo.get_file_line(id), code, id.name.as_str()));
                        } else {
                            println!("cannot find id {:?} when writing files, just ignore it", id);
                            continue;
                        }
                    }
                    if fcodes.len() == 0 {
                        continue;
                    }
                    // merge into one file
                    let cs = merge_rust_codes(fcodes, &main_root, &cargo.id);
                    // cache it
                    cache.files.insert(fkey, cs.clone());
                    cache.save_to_cache();
                    cs
                } else {
                    continue;
                };

                // file path
                let fpath = pdir.join(Path::new(&format!("{}.rs", file)));
                let mut fp = String::new();

                // declare the file mod in package's mod.rs
                pfile.0.push_str(&format!(
                    "mod {};\n",
                    avoid_rust_keywords(file).unwrap_or(file.to_string())
                ));

                // import siblings' mod into the current file
                for (ff, _) in &files {
                    if ff != f {
                        let fmod = ff.strip_suffix(".go").unwrap_or(ff);
                        let fc = avoid_rust_keywords(fmod).unwrap_or(fmod.to_string());
                        fp.push_str(&format!("use super::{}::*;\n", fc));
                    }
                }

                // write imports into the current file
                if let Some(impts) = &merged.imports {
                    let bytes = impts
                        .iter()
                        .fold(String::new(), |acc, imp| format!("{}\n{}", acc, imp));
                    fp.push_str(&bytes);
                }
                fp.push_str("\n");

                // write codes
                fp.push_str(&merged.code);

                // deduplicate by LLM
                // let fp = llm_rust_merge(fp, fpath.as_os_str().to_str().unwrap()).await;

                // persistent
                println!("write file {}", fpath.as_os_str().to_str().unwrap());
                fs::write(&fpath, fp).unwrap();

                // export file symbol onto package's mod.rs
                for id in ids {
                    // must be public and non-method
                    if !pfile.1 && repo.is_exported(id) && !id.name.contains(".") {
                        let (name, _, _) = go_id_to_rust_id(id, repo);
                        pfile.0.push_str(&format!(
                            "pub use self::{}::{};\n",
                            avoid_rust_keywords(file).unwrap_or(file.to_string()),
                            name
                        ));
                    }
                }
            }

            // refresh package's mod.rs
            insert_to_mod_file(pkg_path, pfile.0.as_str(), Some(pfile.1), pkg_files.clone());
        }

        // write pkg mod file
        for (pkg_path, files) in pkg_files.lock().unwrap().iter() {
            let pdir = mdir.join(Path::new(pkg_path));
            if files.1 {
                // inject cargo bin
                cargo.bin(
                    &pkg_path.replace("/", "_"),
                    pdir.strip_prefix(&root_dir).unwrap().to_str().unwrap(),
                );

                // main.rs already exist, just insert mod declaration
                let fpath = pdir.join(Path::new("main.rs"));
                let mut main_file = String::new();
                main_file.push_str(files.0.as_str());
                main_file.push_str(&fs::read_to_string(fpath.clone()).unwrap_or_default());
                fs::write(&fpath, main_file).unwrap();

                // modify root files path
                if pkg_path == "" {
                    root_files[0] = fpath;
                } else {
                    root_files.push(fpath);
                }
            } else if pkg_path == "" && !external_mod {
                // new lib.rs in root dir
                let fpath = pdir.join(Path::new("lib.rs"));
                fs::write(&fpath, files.0.as_str()).unwrap();
            } else {
                // new mod.rs
                let fpath = pdir.join(Path::new("mod.rs"));
                fs::write(&fpath, files.0.as_str()).unwrap();
            }
        }

        // insert into external_mocks/mod.rs
        if external_mod {
            let em_file = root_dir
                .join(Path::new(DIR_EXTERN_MOCKS))
                .join(Path::new("mod.rs"));
            let mut em_cont = fs::read_to_string(&em_file).unwrap_or_default();
            let _mod = format!("pub mod {};\n", &convert_crate(&_mod.name));
            if !em_cont.contains(&_mod) {
                em_cont.push_str(&_mod);
            }
            fs::write(&em_file, em_cont).unwrap();
        }

        // check if root file exists
        let lib = &root_files[0];
        let is_main_lib = lib.file_name().unwrap() == "main.rs";
        let mut bs = if let Ok(bs) = fs::read(lib) {
            String::from_utf8(bs).unwrap()
        } else {
            String::new()
        };
        // add miss mod to bs
        for de in fs::read_dir(&root_dir).unwrap() {
            let path = de.unwrap().path();
            if path.is_dir() {
                // contains mod.rs
                if fs::read_dir(&path)
                    .unwrap()
                    .any(|f| f.unwrap().path().file_name().unwrap() == "mod.rs")
                {
                    let mod_name = path.file_name().unwrap().to_str().unwrap();
                    if !bs.contains(&format!(
                        "{}mod {};\n",
                        if !is_main_lib { "pub " } else { "" },
                        mod_name
                    )) {
                        bs.push_str(&format!(
                            "{}mod {};\n",
                            if !is_main_lib { "pub " } else { "" },
                            mod_name
                        ));
                    }
                }
            }
        }
        fs::write(lib, bs).unwrap();

        // remove LLM produced self crate
        cargo.undep(&convert_crate(&_mod.name));
    }

    // write Cargo.toml file
    fs::write(
        Path::new(dir).join(Path::new("Cargo.toml")),
        cargo.to_string(),
    )
    .unwrap();

    // fmt
    recurse_fmt_files(&root_dir).await;

    // validate all root files
    // for file in &root_files {
    //     llm_rust_validate(file.as_os_str().to_str().unwrap(), 1).await;
    // }

    true
}

#[async_recursion]
async fn recurse_fmt_files(root: &PathBuf) {
    // rustfmt all files
    for dir in fs::read_dir(root).unwrap() {
        let path = dir.unwrap().path();
        if path.is_file() && path.extension().unwrap_or_default() == "rs" {
            llm_rust_fmt(path.as_os_str().to_str().unwrap(), 0).await;
        } else if path.is_dir() {
            recurse_fmt_files(&path).await;
        }
    }
}

// convert go id name to rust id name
//   - name: UpperCamel for type, snake_case for func, UPPER_SNAKE for var
//   - type: UpperCamel
//   - kind: NodeKind
fn go_id_to_rust_id(id: &Identity, repo: &Repository) -> (String, Option<String>, NodeKind) {
    let ids: Vec<&str> = id.name.as_str().split(".").collect();
    let kind = repo.get_kind(id);
    let mut name = match &kind {
        NodeKind::Func => camel_to_snake(ids.last().unwrap()),
        NodeKind::Type => snake_to_camel(ids.last().unwrap()),
        NodeKind::Var => camel_to_snake(ids.last().unwrap()).to_ascii_uppercase(),
        NodeKind::Unknown => camel_to_snake(ids.last().unwrap()),
    };
    if let Some(n) = avoid_rust_keywords(&name) {
        name = n;
    }
    if ids.len() == 2 {
        (name, Some(snake_to_camel(ids[0])), kind)
    } else if ids.len() == 1 {
        (name, None, kind)
    } else {
        panic!("cannot convert go id to rust id: {:?}", id);
    }
}

const MAX_RUST_REFS: usize = 3;

const CONV_RETRY_PROMPT: &str =
    "Cannot find the expected Rust item named '{1}', maybe you recoginize it as wrong kind (should be {2}), please use the name to output again!";

async fn llm_go2rust_convert_core(
    rust_id: &String,
    rust_type: &Option<String>,
    go_new_type: &Option<String>,
    kind: &NodeKind,
    to_convert: String,
    node: &Node,
) -> (String, Option<String>, HashSet<String>, Option<String>) {
    println!("[llm_go2rust_id] start converting: node {:?}", node.id);

    // call llm
    let conv = ToCompress::ToCovertGo2Rust(to_convert.clone());
    let data = compress(&conv).await;
    let codes = extract_code_and_deps(data, false);
    if codes.0 == "" {
        panic!("[WARNING] convert node {:?} failed", node.id);
    }

    let mut imps = HashSet::new();
    let mut sub = "".to_string();

    // parse return Rust codes to AST
    let mut ok = false;
    let mut err = None;
    match syn::parse_file(codes.0.as_str()) {
        Ok(file) => {
            // find id item
            for item in &file.items {
                // println!("item: {}", quote::quote! {#item});
                let (s, found) =
                    collect_rust_item(item, node, rust_id, rust_type, go_new_type, kind, &mut imps);
                if found {
                    ok = true;
                    sub = s;
                    break;
                }
            }
        }
        Err(e) => {
            err = Some(format!(
                "Fail parse rust codes `{}`, err: '{}'",
                codes.0.as_str(),
                e.to_string()
            ));
        }
    }

    if !ok {
        // still save all codes
        sub = codes.0;
        // not found, retry with prompt
        if err.is_none() {
            err = Some(format!(
                "{}: ```json\n{}\n```",
                CONV_RETRY_PROMPT
                    .replace("{1}", &rust_id)
                    .replace("{2}", &kind.to_string()),
                to_convert,
            ))
        }
    }

    (sub, codes.1, imps, err)
}

#[async_recursion]
async fn llm_go2rust_convert<'a>(
    node: &'a Node,
    repo: &'a Repository,
    cache: &mut CodeCache,
    visited: &mut HashSet<&'a Identity>,
) -> Option<Code> {
    // check circular dependency
    if let Some(_) = visited.get(&node.id) {
        println!("[WARNING] circular dependency detected: {:?}", node.id);
        return None;
    } else {
        visited.insert(&node.id);
    }

    let mut content = repo.get_id_content(&node.id).unwrap_or_else(|| {
        // NOTICE: if cannot find the node, it must be a method now
        let ids: Vec<&str> = node.id.name.split(".").collect();
        if ids.len() != 2 {
            panic!("cannot find the node: {:?}", node.id);
        }
        // thus we need to convert it to a method mock
        format!(
            "func({}) {}() {{ todo!(\"Not implemented yet!\"); }}",
            ids[0], ids[1],
        )
    });

    let (rust_id, rust_type, kind) = go_id_to_rust_id(&node.id, repo);
    if !node.id.inside() {
        // mock symbol of FuncType doesn't need func body
        if let NodeKind::Func = kind {
            content = content.split("\n").nth(0).unwrap_or(&content).to_string();
            content += "\n\ttodo!(\"Not implemented yet!\");\n}";
        }
    }

    // fill ToCovert
    let mut to_covert = ToConvert {
        name: rust_id.clone(),
        receiver: rust_type.clone().map(|t| t.to_string()),
        definition: content,
        dependencies: HashMap::new(),
        references: HashMap::new(),
    };

    // NOTICE: LLM always produce `impl {type} { fn new() }` for `new_{type}`
    // thus we must extract it
    let go_new_type = if rust_id.starts_with("new_") {
        Some(snake_to_camel(rust_id.strip_prefix("new_").unwrap()))
    } else {
        None
    };

    // collect deps
    for (dep) in &node.dependencies {
        // get coverted rust codes
        let code = if let Some(cc) = cache.get_by_id(dep) {
            // cached
            cc.code.clone()
        } else {
            let n = repo.get_node(dep);
            if n.is_none() {
                // no codes, just igore
                continue;
            }
            if let NodeKind::Unknown = repo.get_kind(dep) {
                // unknown kind, just igore
                continue;
            }
            if let Some(c) = llm_go2rust_convert(n.unwrap(), repo, cache, visited).await {
                // recurse
                c.code
            } else {
                // failed, just keep go codes
                repo.get_id_content(dep).unwrap_or_default()
            }
        };
        // NOTICE: to reduce the size of the code, only extract the first line of func
        let (rname, _, _) = go_id_to_rust_id(dep, repo);
        let mut def_code = Reference {
            // need_mock: dep.mod_path != node.id.mod_path,
            name: Some(rname),
            code: code,
            import: None,
        };
        if dep.pkg_path != node.id.pkg_path {
            def_code.import = Some(go_impt_to_rust_impt(dep, repo))
        }
        to_covert.dependencies.insert(dep.name.clone(), def_code);
    }

    // collect references
    let mut count = 0;
    for (refer) in &node.references {
        if count > MAX_RUST_REFS {
            println!(
                "[WARNING] too many references for {:?}, only convert first {}",
                node.id.name, MAX_RUST_REFS
            );
            break;
        }
        if let Some(codes) = repo.get_id_content(refer) {
            // NOTICE: to reduce the size of the code, only extract MAX_RUST_REFS refers
            let code = Reference {
                // need_mock: false,
                name: None,
                code: codes,
                import: None,
            };
            to_covert.references.insert(node.id.name.clone(), code);
            count += 1;
        }
    }

    // call LLM and collect rust item
    let prompt = serde_json::to_string(&to_covert).unwrap();
    let (mut codes, mut crates, mut imps, ok) =
        llm_go2rust_convert_core(&rust_id, &rust_type, &go_new_type, &kind, prompt, node).await;
    if let Some(err) = ok {
        // get rust item failed, retry once...
        (codes, crates, imps, _) =
            llm_go2rust_convert_core(&rust_id, &rust_type, &go_new_type, &kind, err, node).await;
    }

    // fiter unneccessary imports
    imps = imps
        .iter()
        .filter(|i| -> bool {
            // keep external import
            if !i.starts_with("use crate ::") {
                true
            } else {
                // only keep internal import that is in the dependencies' package
                let impt = i.strip_prefix("use ").unwrap();
                let impt = impt.strip_suffix(";").unwrap_or(impt).replace(" ", "");
                for dep in to_covert.dependencies.values() {
                    if let Some(pkg) = dep.import.as_deref() {
                        if impt.starts_with(pkg) {
                            return true;
                        }
                    }
                }
                false
            }
        })
        .cloned()
        .collect();

    // insert convert.dependency.imports
    for dep in to_covert.dependencies.values() {
        if let Some(im) = &dep.import {
            imps.insert(format!("use {};", im));
        }
    }

    let code = Code {
        code: codes,
        imports: if imps.len() > 0 { Some(imps) } else { None },
        crates: crates,
    };

    // update cache
    cache.insert_by_id(&node.id, code.clone());
    cache.save_to_cache();

    return Some(code);
}

pub fn collect_rust_item(
    item: &Item,
    node: &Node,
    rust_id: &String,
    rust_type: &Option<String>,
    go_new_type: &Option<String>,
    kind: &NodeKind,
    imps: &mut HashSet<String>,
) -> (String, bool) {
    match item {
        syn::Item::Macro(m) => {
            println!("macro {}", m.mac.path.segments[0].ident);
            if let Some(id) = m.mac.path.segments.last() {
                if rust_id == "init" && id.ident == "lazy_static" {
                    // NOTICE: sometime `init()` will be converted to `lazy_static!` macro
                    return (quote::quote! {#item}.to_string(), true);
                }
            }
            for t in m.mac.tokens.clone().into_iter() {
                if &t.to_string() == rust_id {
                    // FIXME: there may be multiple tokens same with the rust_id
                    return (quote::quote! {#item}.to_string(), true);
                }
            }
            let tt = syn::parse_str::<Item>(&m.mac.tokens.to_string());
            if let Ok(tt) = tt {
                let (_, ok) =
                    collect_rust_item(&tt, node, rust_id, rust_type, go_new_type, kind, imps);
                if ok {
                    return (quote::quote! {#item}.to_string(), true);
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Fn(f) => {
            if f.sig.ident == rust_id {
                if let NodeKind::Func = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got func",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Impl(i) => {
            // get `impl {type} { ... }` type
            let given_type = match i.self_ty.as_ref() {
                syn::Type::Path(p) => {
                    let seg = p.path.segments.last().unwrap();
                    seg.ident.to_string()
                }
                _ => panic!("unrecognized impl type {}", quote::quote! {#i}.to_string()),
            };
            let the_type = if given_type == rust_type.as_deref().unwrap_or_default() {
                rust_type.clone().unwrap()
            } else if given_type == go_new_type.as_deref().unwrap_or_default() {
                go_new_type.clone().unwrap()
            } else {
                "".to_string()
            };
            if the_type != "" {
                let mut the_item = None;
                for tt in &i.items {
                    match tt {
                        syn::ImplItem::Fn(m) => {
                            if m.sig.ident == rust_id
                                || (go_new_type.is_some() && m.sig.ident == "new")
                            {
                                // only reserve the matched method
                                if let NodeKind::Func = kind {
                                    the_item = Some(tt.clone());
                                } else {
                                    println!(
                                        "[WARNING] mismatched kind for {:?}: expect {:?}, got method",
                                        node.id.name, kind,
                                    );
                                }
                            }
                        }
                        _ => continue,
                    }
                }
                if let Some(the_item) = the_item {
                    // filter all other items in impl
                    let mut i = i.clone();
                    i.items = vec![the_item];
                    return (quote::quote! {#i}.to_string(), true);
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Struct(s) => {
            if s.ident == rust_id {
                if let NodeKind::Type = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got struct",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Type(t) => {
            if t.ident == rust_id {
                if let NodeKind::Type = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got type",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Trait(t) => {
            if t.ident == rust_id {
                if let NodeKind::Type = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got trait",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Static(s) => {
            if s.ident == rust_id {
                if let NodeKind::Var = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got static",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Const(c) => {
            if c.ident == rust_id {
                if let NodeKind::Var = kind {
                    return (quote::quote! {#item}.to_string(), true);
                } else {
                    println!(
                        "[WARNING] mismatched kind for {:?}: expect {:?}, got const",
                        node.id.name, kind,
                    );
                }
            }
            ("".to_string(), false)
        }
        syn::Item::Use(_) => {
            // collect all imports
            imps.insert(quote::quote! {#item}.to_string());
            return ("".to_string(), false);
        }
        _ => {
            println!(
                "[WARNING] unrecoginized rust item `{}` for {:?}",
                quote::quote! {#item},
                node.id
            );
            return ("".to_string(), false);
        }
    }
}

async fn llm_rust_merge(codes: String, file: &str) -> String {
    if codes.len() > MAX_LLM_FILE_SIZE {
        return codes;
    }
    println!("[llm_rust_merge] start merge package: {}", file);
    let conv = ToCompress::ToMergeRustPkg(codes);
    let data = compress(&conv).await;
    println!("[llm_rust_merge] succeed merge package: {}", file);
    let codes = extract_code_and_deps(data, false);
    return codes.0;
}

fn merge_rust_codes(
    codes: Vec<(FileLine, &Code, &str)>,
    root: &Option<String>,
    repo_id: &String,
) -> Code {
    // sort by line number
    let mut codes = codes;
    codes.sort_by(|a, b| a.0.line.cmp(&b.0.line));

    // replace imports
    let mut imps: HashSet<String> = HashSet::new();
    for code in &codes {
        if code.1.imports.is_some() {
            for imp in code.1.imports.as_ref().unwrap() {
                let imp = replace_impt_crate(imp, root, repo_id);
                imps.insert(imp);
            }
        }
    }

    let mut js = String::new();
    for code in &codes {
        js.push_str("\n");
        js.push_str(
            format!(
                "// {}/{}#{}\n", // comment origin golang id and codes
                code.0.pkg, code.0.file, code.2
            )
            .as_str(),
        );
        js.push_str(code.1.code.as_str());
        js.push_str("\n");
    }
    Code {
        imports: if imps.len() > 0 { Some(imps) } else { None },
        code: js,
        crates: None, // NOTICE: no need to merge crates for a pkg, because it will always be merged by module
    }
}

async fn llm_rust_fmt(root: &str, repeat: usize) {
    match cmd::run_command("rustfmt", vec!["--edition=2021", root]) {
        Ok(_) => return,
        Err(e) => {
            println!(
                "[llm_rust_validate] fail to rustfmt file {} for {} time, err {}, retry...",
                root, -1, e
            );
        }
    }
    // format
    for i in 0..repeat {
        // cmd call `rustfmt --edition=2021 {file_path}`
        match cmd::run_command("rustfmt", vec!["--edition=2021", root]) {
            Ok(_) => return,
            Err(e) => {
                println!(
                    "[llm_rust_validate] fail to rustfmt file {} for {} time, err {}, retry...",
                    root, i, e
                );
                modify_file_with_error(&e.to_string(), false).await;
            }
        }
    }
}

async fn llm_rust_validate(root: &str, repeat: usize) {
    // loop calling at most rustc times...
    for i in 0..repeat {
        // cmd call `rustc --emit=metadata {file_path}`
        match cmd::run_command("rustc", vec!["--emit=metadata", root]) {
            Ok(_) => break,
            Err(e) => {
                println!(
                    "[llm_rust_validate] validate package {} for {} time, err {}",
                    root, i, &e,
                );
                modify_file_with_error(&e.to_string(), true).await;
            }
        }
    }
}

async fn modify_file_with_error(err: &str, ignore: bool) {
    // extra error files
    let files = extract_msg_from_err(err, ignore);
    for (file_path, tips) in files {
        //read from file
        let code = fs::read(file_path).unwrap();
        if code.len() > MAX_LLM_FILE_SIZE {
            println!(
                "[WARNING] file size too large, skip modify file: {}",
                file_path
            );
            continue;
        }
        let js = to_string(&ToValidate {
            code: String::from_utf8(code).unwrap(),
            error: tips,
        })
        .unwrap();
        let conv = ToCompress::ToValidateRust(js);
        let data = compress(&conv).await;
        let codes = extract_code_and_deps(data, true);
        // write to file
        fs::write(file_path, codes.0).unwrap();
    }
}

// extract '```rust\n' and '\n```'
// biggest indicates only extract the biggest code chunk, otherwise merge all code chunks
fn extract_code_and_deps(data: String, biggest: bool) -> (String, Option<String>) {
    let codes = data.split("```").collect::<Vec<&str>>();
    if codes.len() < 2 {
        panic!("cannot extract codes from LLM result: {:?}", data);
    }
    let (mut code, mut dep) = (String::new(), None);

    for i in 1..codes.len() {
        let c = codes[i];
        if c.contains("toml\n") && c.contains("[dependencies]") {
            dep = Some(c.strip_prefix("toml\n").unwrap().to_string());
        } else if c.contains("rust\n") {
            let tmp = c.strip_prefix("rust").unwrap();
            if biggest {
                if tmp.len() > code.len() {
                    code = tmp.to_string();
                }
            } else {
                code.push_str(tmp);
            }
        }
    }

    (code, dep)
}
