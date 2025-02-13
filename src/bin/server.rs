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
use std::fs::File;
use std::io::Write;
use std::net::SocketAddr;
// Import necessary types from the standard library
use std::path::{Path, PathBuf};

use std::sync::Arc;

use futures::future::BoxFuture;
use futures::FutureExt;
use hyper::{Body, StatusCode};
use ryze::hertz::{Hertz, RequestContext};
use serde::{Deserialize, Serialize};

// use crate::llm::llm;
use ABCoder::compress::compress::{compress_all, from_json};
use ABCoder::compress::conv::convert_go2rust;
use ABCoder::compress::types::types::{Code, CodeCache, Identity};

use ABCoder::config::{self, CONFIG};
use ABCoder::storage::cache::{self, get_cache};
use ABCoder::utils::cmd;
use ABCoder::utils::files;
// Import the git module
use ABCoder::utils::git;
use ABCoder::utils::git::RepositoryStat;
use ABCoder::utils::markdown;

#[derive(Serialize, Deserialize, Debug)]
pub struct BasicInfo {
    readme: String,
    repo_stats: RepositoryStat,
}

// basic_info handler
fn basic_info(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap();

    println!("{}", repo);

    let repo_dir = check_repo_exist(&repo);

    if let Some(body) = markdown::get_readme_json(Path::new(repo_dir.as_str())) {
        let repo = repo.clone();
        return (async move {
            match git::get_repo_stats(repo.as_str()).await {
                Ok(repo) => {
                    println!("Successfully fetched repo stats");
                    let mut basic_info = BasicInfo {
                        readme: body,
                        repo_stats: repo,
                    };
                    let body = serde_json::to_string_pretty(&basic_info).unwrap();
                    *ctx.resp.body_mut() = Body::from(body);
                }
                Err(e) => eprintln!("Failed to fetch repo stats: {}", e),
            };
        })
        .boxed();
    }

    *ctx.resp.body_mut() = Body::from("Not found README.md in this project.");
    return (async move {}).boxed();
}

// repo_stats handler
fn repo_stats(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    println!("{}", repo);

    (async move {
        match git::get_repo_stats(repo.as_str()).await {
            Ok(repo) => {
                println!("Successfully fetched repo stats");
                let body = serde_json::to_string_pretty(&repo).unwrap();
                *ctx.resp.body_mut() = Body::from(body);
            }
            Err(e) => eprintln!("Failed to fetch repo stats: {}", e),
        };
    })
    .boxed()
}

fn check_repo_exist(repo: &String) -> String {
    // Url of the repository to be cloned
    let mut base_repo = repo.clone();
    // sub git module
    if repo.split("/").count() > 2 {
        let parts: Vec<&str> = repo.split("/").take(2).collect();
        base_repo = parts.join("/");
    }
    let repo_url = format!("git@github.com:{}.git", base_repo);
    println!("{}", repo_url);
    // Directory where you want to clone the repository
    let repo_dir_path = Path::new(&CONFIG.repo_dir).join(&base_repo);
    if !repo_dir_path.exists() {
        // Call the git_clone function and handle any errors that occur
        match git::git_clone(repo_url.as_str(), repo_dir_path.as_path()) {
            Ok(()) => println!("Git repo cloned successfully!"),
            Err(e) => eprintln!("An error occurred while cloning the repo: {}", e),
        }
    } else {
        println!("Already cloned: {}", base_repo)
    }

    // return the real repo dir - submodule case
    repo_dir_path.to_str().unwrap().to_string()
}

// code_analyze handler
fn code_analyze(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    let update_ast = parsed.get("merge").is_some();
    let id = parsed.get("id").unwrap_or(&"".to_string()).clone();
    println!("repo: {}, id: {}, merge: {}", repo, id, update_ast);
    (async move {
        let repo_dir = check_repo_exist(&repo);
        let mut repo_c = get_concat(&repo, Some(&id));
        if update_ast || repo_c.is_none() {
            match cmd::run_command(&config::go_ast_path(), vec![repo_dir.as_str(), &id]) {
                Ok(mut output) => {
                    println!("parse repo successfull, output: {}", output);
                    if repo_c.is_some() {
                        // merge
                        let mut old =
                            from_json(&repo, String::from_utf8(repo_c.unwrap()).unwrap().as_str())
                                .unwrap();
                        let rep = from_json(&repo, output.as_str()).unwrap();
                        old.merge_with(&rep);
                        output = serde_json::to_string(&old).unwrap();
                        set_concat(&repo, Some(&id), Vec::from(output.clone()));
                    } else {
                        from_json(&repo, output.as_str()).unwrap();
                    }
                    repo_c = Some(Vec::from(output));
                }
                Err(err) => {
                    eprint!("plugin parse repo {} error: {:?}", repo_dir.as_str(), err);
                    *ctx.resp.status_mut() = StatusCode::INTERNAL_SERVER_ERROR;
                    return;
                }
            }
        }
        // return repo_c
        if let Some(repo_c) = repo_c {
            *ctx.resp.body_mut() = Body::from(repo_c);
        }
        return;
    })
    .boxed()
}

fn repo_go_to_rust(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    println!("[repo_go2rust] repo: {}", repo);
    (async move {
        let repo_dir = check_repo_exist(&repo);
        let mut repo_c = get(&repo);
        if repo_c.is_none() {
            match cmd::run_command(
                &config::go_ast_path(),
                vec!["--refer_code_depth=1", repo_dir.as_str()],
            ) {
                Ok(output) => {
                    println!("[repo_go2rust] parse repo successfull, output: {}", output);
                    repo_c = Some(Vec::from(output));
                }
                Err(err) => {
                    eprint!(
                        "[repo_go2rust] plugin parse repo {} error: {:?}",
                        repo_dir.as_str(),
                        err
                    );
                    *ctx.resp.status_mut() = StatusCode::INTERNAL_SERVER_ERROR;
                    return;
                }
            }
        }
        let repo_c = repo_c.unwrap();
        let mut ast = from_json(&repo, String::from_utf8(repo_c).unwrap().as_str()).unwrap();
        let mut m = CodeCache::new(format!("go2rust_{}", ast.id));
        m.load_from_cache();
        let out_dir = Path::new(&CONFIG.repo_dir).join("go2rust").join(&repo);
        convert_go2rust(&mut ast, out_dir.to_str().unwrap(), &mut m).await;
        *ctx.resp.body_mut() = Body::from("OK");
        return;
    })
    .boxed()
}

// issue_trace handler
fn issue_trace(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();

    (async move {
        let mut body = String::from("please pass the key words through query key issue");

        if let Some(issue_key_words) = parsed.get("issue") {
            let mut issues = git::search_issue(repo.as_str(), issue_key_words.as_str(), 3)
                .await
                .unwrap();
            body = serde_json::to_string_pretty(&issues).unwrap();
        }
        *ctx.resp.body_mut() = Body::from(body);
    })
    .boxed()
}

#[derive(Serialize, Deserialize, Debug)]
pub struct TreeStructure {
    tree: String,
}

// tree_structure handler
fn tree_structure(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    (async move {
        check_repo_exist(&repo);
        let suffix = "go";
        let mut body = String::from("generate tree failed.");
        let repo = Path::new(&CONFIG.repo_dir).join(&repo);
        let mut tree_struct = TreeStructure {
            tree: "".to_string(),
        };
        if let Some(tree) = files::tree(&repo, suffix) {
            tree_struct.tree = tree.to_string();
            body = serde_json::to_string_pretty(&tree_struct).unwrap();
        }
        *ctx.resp.body_mut() = Body::from(body);
    })
    .boxed()
}

pub fn get(key: &String) -> Option<Vec<u8>> {
    let mut cache = cache::get_cache();
    cache.get(key)
}

pub fn get_concat(key: &String, id: Option<&String>) -> Option<Vec<u8>> {
    let mut cache = cache::get_cache();
    if id.is_some() {
        cache.get(format!("{}?{}", key, id.unwrap()).as_str())
    } else {
        None
    }
}

pub fn set_concat(key: &String, id: Option<&String>, entity: Vec<u8>) {
    let mut cache = cache::get_cache();
    if id.is_some() {
        cache
            .put(format!("{}?{}", key, id.unwrap()).as_str(), entity)
            .unwrap();
    } else {
        cache.put(key.as_str(), entity).unwrap();
    }
}

// #[tokio::main]
fn main() {
    let rt = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(4)
        .thread_stack_size(32 * 1024 * 1024) // Use 32MB per worker thread
        .enable_all()
        .build()
        .unwrap();

    rt.block_on(async {
        let h = Hertz::new();
        h.get("/basic_info", Arc::new(basic_info)).await;
        h.get("/repo_stats", Arc::new(repo_stats)).await;

        h.get("/issue_trace", Arc::new(issue_trace)).await;
        h.get("/repo_structure", Arc::new(tree_structure)).await;
        h.get("/code_analyze", Arc::new(code_analyze)).await;

        h.get("/repo_go2rust", Arc::new(repo_go_to_rust)).await;

        h.spin(SocketAddr::from(([0, 0, 0, 0, 0, 0, 0, 0], 8888)))
            .await
            .expect("TODO: panic message");
    });
}
