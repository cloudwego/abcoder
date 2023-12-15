use std::fs::File;
use std::io::Read;
use std::io::Write;
use std::ops::Add;
use std::path::Path;
use std::process::Command;

use reqwest::Client;
use reqwest::Error as RError;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use serde_yaml;
use tokio;

use super::errors::Error;

// TODO config-lize
static GLOBAL_GIT_TOKEN: Option<&str> = option_env!("ABCoder_github_token");

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
pub struct Repository {
    stargazers_count: usize,
    forks: usize,
    open_issues: usize,
}

pub async fn get_repo_stats(repo: &str) -> Result<(Repository), Box<dyn std::error::Error>> {
    let request_url = format!("https://api.github.com/repos/{}", repo);
    let client = reqwest::Client::new();
    let response = client
        .get(&request_url)
        .header("User-Agent", "reqwest")
        .header(
            "Authorization",
            "Bearer ".to_string().add(GLOBAL_GIT_TOKEN.unwrap()),
        )
        .send()
        .await?;

    if response.status() != 200 {
        return Err(Box::new(std::io::Error::new(std::io::ErrorKind::Other,
                                                format!("status is not 200, body is: {}", response.text().await?))));
    }

    // println!("{}",response.text().await?);

    let resp: Repository = response.json().await?;

    println!("Stars: {}", resp.stargazers_count);
    println!("Forks: {}", resp.forks);
    println!("Open Issues: {}", resp.open_issues);

    Ok((resp))
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Issue {
    pub title: String,
    pub body: String,
    pub url: String,
    pub isClosed: bool,
}


pub async fn search_issue(
    repo: &str,
    keywords: &str, limit: i8,
) -> Result<Vec<Issue>, Box<dyn std::error::Error>> {
    let client = Client::new();

    let resp = client
        .get(format!(
            "https://api.github.com/search/issues?q=repo:{}+is:issue+{}",
            repo, keywords
        ))
        .header("User-Agent", "Your-User-Agent") // Replace "Your-User-Agent" with the actual one
        .header(
            "Authorization",
            "Bearer ".to_string().add(GLOBAL_GIT_TOKEN.unwrap()),
        )
        .send()
        .await?;

    let body = resp.text().await?;

    let data: Value = serde_json::from_str(&body)?;
    let issues_data = data["items"].as_array().unwrap();

    let mut issues = vec![];

    let mut issue_count = 0;

    for item in issues_data {
        if limit != -1 && issue_count >= limit {
            break;
        }

        let issue = Issue {
            title: item["title"].as_str().unwrap().to_string(),
            body: item["body"].as_str().unwrap().to_string(),
            url: item["html_url"].as_str().unwrap().to_string(),
            isClosed: item["state"].as_str().unwrap().eq("closed"),
        };

        issues.push(issue);
        issue_count += 1;
    }

    // let doc_content = serde_yaml::to_string(&issues)?;

    // let mut file = File::create("issues.yml")?;
    // file.write_all(doc_content.as_bytes())?;

    println!("Done");
    Ok(issues)
}
