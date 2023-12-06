// Import necessary types from the standard library
use std::path::Path;

// Import the git module
// use crate::utils::git;
use crate::utils::markdown;

mod utils;

fn main() {
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

    println!("{:?}", md_list);
}