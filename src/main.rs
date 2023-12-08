// Import necessary types from the standard library
use std::path::Path;
use std::ptr::addr_of_mut;

use tokio::runtime::Runtime;

// use crate::llm::llm;
use utils::llm::count_tokens_rough;

use crate::utils::files;
// Import the git module
use crate::utils::git;
use crate::utils::markdown;

mod utils;
mod compress;

#[tokio::main]
async fn main() {
    // Url of the repository to be cloned
    let repo_url = "https://github.com/cloudwego/hertz.git";

    // Directory where you want to clone the repository
    let repo_dir = Path::new("./tmp/hertz");

    // Call the git_clone function and handle any errors that occur
    match git::git_clone(repo_url, repo_dir) {
        Ok(()) => println!("Git repo cloned successfully!"),
        Err(e) => eprintln!("An error occurred while cloning the repo: {}", e),
    }


    let mut md_list = Vec::new();
    markdown::get_all_md(repo_dir, &mut md_list).expect("TODO: panic message");

    for md in &md_list {
        println!("{}:{:?}", md, files::count_lines(Path::new(md.as_str()), true).unwrap())
    }
    for md in &md_list {
        println!("{}:{:?}", md, files::count_lines(Path::new(md.as_str()), false).unwrap())
    }

    for md in &md_list {
        println!("{}:{:?}", md, count_tokens_rough(Path::new(md.as_str())).unwrap())
    }


    match git::get_repo_stats("cloudwego", "hertz").await {
        Ok(_) => println!("Successfully fetched repo stats"),
        Err(e) => eprintln!("Failed to fetch repo stats: {}", e),
    }

    git::search_issue("cloudwego", "hertz","server closed connection").await;
}