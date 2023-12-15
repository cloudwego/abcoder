use std::arch::asm;
use std::net::SocketAddr;
// Import necessary types from the standard library
use std::path::{Path, PathBuf};
use std::ptr::addr_of_mut;
use std::sync::Arc;

use futures::future::BoxFuture;
use futures::FutureExt;
use hyper::Body;
use hyper::body::HttpBody;
use reqwest::Url;
use ryze::hertz::{Hertz, RequestContext};
use serde::{Deserialize, Serialize};
use tokio::runtime::Runtime;

// use crate::llm::llm;
use utils::llm::count_tokens_rough;

use crate::compress::golang;
use crate::compress::parser::LanguageParser;
use crate::utils::cmd;
use crate::utils::files;
// Import the git module
use crate::utils::git;
use crate::utils::markdown;

mod compress;
mod utils;


// basic_info handler
fn basic_info(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> = serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap();


    println!("{}", repo);

    let repo_dir = check_repo_exist(&repo);

    if let Some(body) = markdown::get_readme_json(Path::new(repo_dir.as_str())) {
        *ctx.resp.body_mut() = Body::from(body);
        return (async move {}).boxed();
    }

    *ctx.resp.body_mut() = Body::from("Not found README.md in this project.");
    return (async move {}).boxed();
}


// repo_stats handler
fn repo_stats(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> = serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
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
    }).boxed()
}

fn check_repo_exist(repo: &String) -> String {
    // Url of the repository to be cloned
    let mut base_repo = repo.clone();
    // sub git module
    if repo.split("/").count() > 2 {
        let parts: Vec<&str> = repo.split("/").take(2).collect();
        base_repo = parts.join("/");
    }
    let repo_url = format!("https://github.com/{}.git", base_repo);
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
    repo_dir
}


// code_analyze handler
fn code_analyze(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> = serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    (async move {
        check_repo_exist(&repo);

        if let Ok(output) = cmd::run_command("./go_ast", vec!["./tmp/welkeyever/hertz/cmd/hz"]) {
            *ctx.resp.body_mut() = Body::from(output);
            return;
        }
        *ctx.resp.body_mut() = Body::from("analyze failed.");
    }).boxed()
}

// issue_trace handler
fn issue_trace(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> = serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();

    (async move {
        let mut body = String::from("please pass the key words through query key： issue");

        if let Some(issue_key_words) = parsed.get("issue") {
            let mut issues = git::search_issue(repo.as_str(), issue_key_words.as_str(), 3).await.unwrap();
            body = serde_json::to_string_pretty(&issues).unwrap();
        }
        *ctx.resp.body_mut() = Body::from(body);
    }).boxed()
}

// tree_structure handler
fn tree_structure(ctx: &mut RequestContext) -> BoxFuture<'_, ()> {
    let parsed: std::collections::HashMap<String, String> = serde_urlencoded::from_str(ctx.req.uri().query().unwrap()).unwrap();
    let repo = parsed.get("repo").unwrap().clone();
    (async move {
        check_repo_exist(&repo);
        let suffix = "go";
        let mut body = String::from("generate tree failed.");
        let repo = PathBuf::from(format!("./tmp/{}", repo));
        if let Some(tree) = files::tree(&repo, suffix) {
            body = tree.to_string()
        }
        *ctx.resp.body_mut() = Body::from(body);
    }).boxed()
}


#[tokio::main]
async fn main() {
    let h = Hertz::new();
    h.get("/basic_info", Arc::new(basic_info)).await;
    h.get("/repo_stats", Arc::new(repo_stats)).await;
    h.get("/issue_trace", Arc::new(issue_trace)).await;

    h.get("/repo_structure", Arc::new(tree_structure)).await;

    h.get("/code_analyze", Arc::new(code_analyze)).await;


    h.spin(SocketAddr::from(([0, 0, 0, 0, 0, 0, 0, 0], 8888))).await.expect("TODO: panic message");

    // Url of the repository to be cloned
    let repo_url = "https://github.com/cloudwego/hertz.git";

    // Directory where you want to clone the repository
    let repo_dir = Path::new("./tmp/hertz");

    // Call the git_clone function and handle any errors that occur
    // match git::git_clone(repo_url, repo_dir) {
    //     Ok(()) => println!("Git repo cloned successfully!"),
    //     Err(e) => eprintln!("An error occurred while cloning the repo: {}", e),
    // }

    let mut md_list = Vec::new();
    markdown::get_all_md(repo_dir, &mut md_list).expect("TODO: panic message");

    for md in &md_list {
        println!(
            "{}:{:?}",
            md,
            files::count_lines(Path::new(md.as_str()), true).unwrap()
        )
    }
    for md in &md_list {
        println!(
            "{}:{:?}",
            md,
            files::count_lines(Path::new(md.as_str()), false).unwrap()
        )
    }

    for md in &md_list {
        println!(
            "{}:{:?}",
            md,
            count_tokens_rough(Path::new(md.as_str())).unwrap()
        )
    }

    // match git::get_repo_stats("cloudwego", "hertz").await {
    //     Ok(_) => println!("Successfully fetched repo stats"),
    //     Err(e) => eprintln!("Failed to fetch repo stats: {}", e),
    // }

    // git::search_issue("cloudwego", "hertz","server closed connection").await;

    // let mut p = golang::parser::GolangParser::new();
    // p.process();
}