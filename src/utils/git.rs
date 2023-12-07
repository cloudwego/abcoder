use std::io::Read;
use std::ops::Add;
use std::path::Path;
use std::process::Command;

use reqwest::Error as RError;
use serde::{Deserialize, Serialize};
use tokio;

use super::errors::Error;

// Function that clones a git repository and takes the URL of the
// repository and the directory where it should be cloned as arguments
pub fn git_clone(url: &str, directory: &Path) -> Result<(), Error> {
    // The git command is executed as a child process
    let output = Command::new("git")
        .args(&["clone", url, directory.to_str().unwrap()])
        .output()?;

    // Check if the git clone operation was successful
    if output.status.success() {
        Ok(())
    } else {
        // Convert the Output into our custom error type and return the error
        Err(output.into())
    }
}


// 以下代码用于描述返回的 json 数据结构
#[derive(Serialize, Deserialize, Debug)]
struct Repository {
    stargazers_count: usize,
    forks: usize,
    open_issues: usize,
}

pub async fn get_repo_stats(user: &str, repo: &str) -> Result<(), RError> {
    let request_url = format!("https://api.github.com/repos/{}/{}", user, repo);
    let client = reqwest::Client::new();

    // TODO config-lize
    let token_value = env!("ABCoder_github_token");

    let response = client
        .get(&request_url)
        .header("User-Agent", "reqwest")
        .header("Authorization", "Bearer ".to_string().add(token_value))
        .send()
        .await?;

    if response.status() != 200 {
        println!("status is not 200, body is: {}", response.text().await?);
        return Ok(());
    }

    let response: Repository = response.json().await?;

    println!("Stars: {}", response.stargazers_count);
    println!("Forks: {}", response.forks);
    println!("Open Issues: {}", response.open_issues);

    Ok(())
}