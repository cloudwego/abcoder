use std::arch::asm;
use std::cell::RefCell;
use std::error::Error;
use std::fs::File;
use std::io::{Stderr, Write};
use std::net::SocketAddr;
use std::os;
// Import necessary types from the standard library
use std::path::{Path, PathBuf};
use std::ptr::addr_of_mut;
use std::rc::Rc;
use std::sync::{Arc, Mutex};

use compress::types::types::Repository;
use futures::future::BoxFuture;
use futures::FutureExt;
use hyper::body::HttpBody;
use hyper::{Body, StatusCode};
use reqwest::Url;
use ryze::hertz::{Hertz, RequestContext};
use serde::{Deserialize, Serialize};
use tokio::runtime::Runtime;

// use crate::llm::llm;
use crate::compress::compress::from_json;
use utils::llm::count_tokens_rough;

use crate::compress::golang;
use crate::compress::parser::LanguageParser;
use crate::storage::cache::{get_cache, CachingStorageEngine, StorageEngine};
use crate::storage::fs::FileStorage;
use crate::storage::memory::MemoryCache;
use crate::storage::{cache, fs, memory};
use crate::utils::cmd;
use crate::utils::files;
// Import the git module
use crate::utils::git;
use crate::utils::git::RepositoryStat;
use crate::utils::markdown;

mod compress;
mod storage;
mod utils;

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

fn check_repo_compress(repo: &String) -> Option<Vec<u8>> {
    let mut cache = cache::get_cache();

    cache.get(repo)
}

fn check_repo_entity(repo: &String, id: Option<&String>) -> Option<Vec<u8>> {
    let mut cache = cache::get_cache();
    if id.is_some() {
        cache.get(format!("{}?{}", repo, id.unwrap()).as_str())
    } else {
        cache.get(repo)
    }
}

fn set_repo_entity(repo: &String, id: Option<&String>, entity: Vec<u8>) {
    let mut cache = cache::get_cache();
    if id.is_some() {
        cache
            .put(format!("{}?{}", repo, id.unwrap()).as_str(), entity)
            .unwrap();
    } else {
        cache.put(repo.as_str(), entity).unwrap();
    }
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
    let repo_dir = format!("./tmp/{}", base_repo);
    // Directory where you want to clone the repository
    let repo_dir_path = Path::new(repo_dir.as_str());
    if !repo_dir_path.exists() {
        // Call the git_clone function and handle any errors that occur
        match git::git_clone(repo_url.as_str(), repo_dir_path) {
            Ok(()) => println!("Git repo cloned successfully!"),
            Err(e) => eprintln!("An error occurred while cloning the repo: {}", e),
        }
    } else {
        println!("Already cloned: {}", base_repo)
    }

    // return the real repo dir - submodule case
    let repo_dir = format!("./tmp/{}", repo);
    repo_dir
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
        let mut repo_c = check_repo_entity(&repo, Some(&id));
        if update_ast || repo_c.is_none() {
            match cmd::run_command("./go_ast", vec![repo_dir.as_str(), &id]) {
                Ok(mut output) => {
                    println!("parse repo successfull, output: {}", output);
                    if repo_c.is_some() {
                        // merge
                        let mut old =
                            from_json(String::from_utf8(repo_c.unwrap()).unwrap().as_str())
                                .unwrap();
                        let rep = from_json(output.as_str()).unwrap();
                        old.merge_with(&rep);
                        old.id = format!("github.com/{}", repo);
                        output = serde_json::to_string(&old).unwrap();
                        set_repo_entity(&repo, Some(&id), Vec::from(output.clone()));
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

// convert repo to csv handler
fn repo_to_csv(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    (async move {
        match check_repo_compress(&repo) {
            Some(re) => {
                println!("start to convert repo: {}", repo);
                let repo =
                    compress::compress::from_json(String::from_utf8(re).unwrap().as_str()).unwrap();
                // write summary to csv
                let csv_sum = repo.to_csv_summary();
                let path_sum = format!("./tmp_compress/{}_summary.csv", repo.id.replace("/", "_"));
                println!("convert repo to csv: {}", path_sum);
                let mut file = File::create(&path_sum).unwrap();
                file.write_all(csv_sum.as_bytes()).unwrap();

                // write summary to csv
                let csv_decl = repo.to_csv_decl();
                let path_decl = format!("./tmp_compress/{}_decl.csv", repo.id.replace("/", "_"));
                println!("convert repo to csv: {}", path_sum);
                let mut file = File::create(&path_decl).unwrap();
                file.write_all(csv_decl.as_bytes()).unwrap();

                // write package to csv
                let csv_pkg = repo.to_csv_pkgs();
                let path_pkg = format!("./tmp_compress/{}_pkg.csv", repo.id.replace("/", "_"));
                println!("convert repo to csv: {}", path_sum);
                let mut file = File::create(&path_pkg).unwrap();
                file.write_all(csv_pkg.as_bytes()).unwrap();

                *ctx.resp.body_mut() = "success".into();
                return;
            }
            _ => {
                println!("no repo parsed");
                *ctx.resp.body_mut() = Body::from("convert failed.");
            }
        }
    })
    .boxed()
}

// repo_compress handler
fn repo_compress(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    (async move {
        let repo_dir = check_repo_exist(&repo);
        let repo_c = check_repo_compress(&repo);
        let mut repo_str = String::new();
        if repo_c.is_none() {
            match cmd::run_command("./go_ast", vec![repo_dir.as_str()]) {
                Ok(output) => {
                    println!("parse repo successfull, output: {}", output);
                    let mut rep = from_json(output.as_str()).unwrap();
                    rep.id = format!("github.com/{}", repo);
                    repo_str = serde_json::to_string(&rep).unwrap();
                    get_cache()
                        .put(repo.as_str(), Vec::from(repo_str.clone()))
                        .unwrap();
                }
                Err(err) => {
                    eprint!(
                        "plugin parse repo {} error: {}",
                        repo_dir.as_str(),
                        err.to_string()
                    );
                    *ctx.resp.status_mut() = StatusCode::INTERNAL_SERVER_ERROR;
                    return;
                }
            }
        } else {
            println!("load compress repo from local:{}", repo.as_str());
            repo_str = String::from_utf8(repo_c.unwrap().clone()).unwrap();
        }

        if !repo_str.is_empty() {
            println!("start to compress repo: {}", repo);
            let mut repo = compress::compress::from_json(repo_str.as_str()).unwrap();
            compress::compress::compress_all(&mut repo).await;
            let compress = serde_json::to_string(&repo).unwrap();
            println!("compressed repo:\n{}", compress);

            *ctx.resp.body_mut() = Body::from(compress);
            return;
        }

        *ctx.resp.body_mut() = Body::from("analyze failed.");
    })
    .boxed()
}

// issue_trace handler
fn issue_trace(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> =
        serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();

    (async move {
        let mut body = String::from("please pass the key words through query key： issue");

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
        let repo = PathBuf::from(format!("./tmp/{}", repo));
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

        h.get("/repo_compress", Arc::new(repo_compress)).await;
        h.get("/repo_to_csv", Arc::new(repo_to_csv)).await;

        h.spin(SocketAddr::from(([0, 0, 0, 0, 0, 0, 0, 0], 8888)))
            .await
            .expect("TODO: panic message");
    });
}
