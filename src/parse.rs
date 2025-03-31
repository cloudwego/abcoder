/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
use std::path::{Path, PathBuf};

use crate::{
    compress::{compress, types::types::Repository},
    config::{self, CONFIG},
    storage::cache,
    utils::{cmd, errors::Error, git, split},
};

#[derive(Clone, Debug, Default)]
pub struct CompressOptions {
    pub parse_only: bool,
    pub not_load_external_symbol: bool,
    pub no_need_comment: bool,
    pub force_update_ast: bool,
}

pub fn force_parse_repo(repo_path: &String, opts: &CompressOptions) -> Result<Repository, Error> {
    let path = parse_repo_path(repo_path)?;
    // force to parse the repo
    let data = parse_repo(&path, opts)?;
    match compress::from_json(&repo_path, String::from_utf8(data).unwrap().as_str()) {
        Ok(repo) => Ok(repo),
        Err(err) => Err(Error::Parse(err.to_string())),
    }
}

fn parse_repo_path(repo_path: &String) -> Result<PathBuf, Error> {
    let git_dir = Path::new(CONFIG.work_dir.as_str());

    let ps: Vec<&str> = repo_path.split('/').collect();
    let path = if repo_path.ends_with(".git") || repo_path.starts_with("https://") {
        // url
        let repo_name = ps[ps.len() - 1].strip_suffix(".git").unwrap();
        let path = git_dir.join(repo_name);
        if !path.exists() {
            // git clone
            git::git_clone(&repo_path, &path).expect("Failed to clone repo");
        }
        path
    } else {
        // existing path
        let path = git_dir.join(&repo_path);
        if !path.exists() {
            println!("path not exists: {:?}", path);
            return Err(Error::GitCloneError("path not exists".to_string()));
        }
        // directly use the repo name
        path
    };
    Ok(path)
}

pub fn get_repo(repo_path: &String, opts: &CompressOptions) -> Result<Repository, Error> {
    let path = parse_repo_path(repo_path)?;

    // check if cache the result
    let data = if let Some(data) = cache::get_cache().get(&path.to_str().unwrap()) {
        data
    } else {
        // parse the repo
        parse_repo(&path, opts)?
    };

    match compress::from_json(&repo_path, String::from_utf8(data).unwrap().as_str()) {
        Ok(repo) => Ok(repo),
        Err(err) => Err(Error::Parse(err.to_string())),
    }
}

fn parse_repo(path: &Path, opts: &CompressOptions) -> Result<Vec<u8>, Error> {
    let (parser, args) = config::parser_and_args(path.to_str().unwrap(), opts);
    // parse the repo by parse
    match cmd::run_command_bytes(&parser, args) {
        Ok(output) => {
            cache::get_cache()
                .put(&path.to_str().unwrap(), output.clone())
                .unwrap();
            return Ok(output);
        }
        Err(err) => {
            println!(
                "plugin parse repo {} error: {}",
                path.to_str().unwrap(),
                err.to_string()
            );
            return Err(Error::Parse(err.to_string()));
        }
    }
}
