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
    fs::{self, File},
    io::Write,
    path::{Path, PathBuf},
};

use csv::Writer;

use crate::{
    compress::{compress, types::types::Repository},
    config::{self, CONFIG},
    storage::cache,
    utils::{cmd, errors::Error, git, split},
};

fn to_csv_summary(repo: &Repository) -> String {
    let mut w = Writer::from_writer(Vec::new());
    // add header
    w.write_record(&["Package", "Name", "Kind", "Signature", "Summary"])
        .unwrap();

    for (mod_name, _mod) in repo.modules.iter() {
        for (pname, pkg) in _mod.packages.iter() {
            for (name, f) in pkg.functions.iter() {
                let tmp = &"".to_string();
                // split content, 1024B for each
                let sums = split::split_text(f.compress_data.as_ref().unwrap_or(tmp), 924);
                for sum in sums {
                    w.write_record(&[
                        pname,
                        name,
                        "Function",
                        f.content.split_once('\n').unwrap_or((&f.content, "")).0,
                        &format!("{}: {}", name, sum),
                    ])
                    .unwrap();
                }
            }
            for (name, t) in pkg.types.iter() {
                let tmp = &"".to_string();
                // split content, 1024B for each
                let sums = split::split_text(t.compress_data.as_ref().unwrap_or(tmp), 924);
                for sum in sums {
                    w.write_record(&[
                        pname,
                        name,
                        "Type",
                        t.content.split_once('\n').unwrap_or((&t.content, "")).0,
                        &format!("{}: {}", name, sum),
                    ])
                    .unwrap();
                }
            }
            for (name, v) in pkg.vars.iter() {
                let tmp = &"".to_string();
                // split content, 1024B for each
                let sums = split::split_text(v.compress_data.as_ref().unwrap_or(tmp), 924);
                for sum in sums {
                    w.write_record(&[
                        pname,
                        name,
                        "Var",
                        v.content.split_once('\n').unwrap_or((&v.content, "")).0,
                        &format!("{}: {}", name, sum),
                    ])
                    .unwrap();
                }
            }
        }
    }
    w.flush().unwrap();
    String::from_utf8(w.into_inner().unwrap()).unwrap()
}

pub fn to_csv_decl(repo: &Repository) -> String {
    let mut w = Writer::from_writer(Vec::new());
    // add header
    w.write_record(&["Identity", "Kind", "Definition"]).unwrap();

    for (mod_name, _mod) in repo.modules.iter() {
        for (pname, pkg) in _mod.packages.iter() {
            for (name, f) in pkg.functions.iter() {
                let decl = f.content.as_str();
                // split content, 1024B for each
                let mut start = 0;
                let mut end = 1024;
                while start < decl.len() {
                    if end > decl.len() {
                        end = decl.len();
                    }
                    if start >= 1024 {
                        start -= 100;
                    }
                    w.write_record(&[
                        &format!("{}.{}", pname, name),
                        "Function",
                        &decl[start..end],
                    ])
                    .unwrap();
                    start = end;
                    end += 924;
                }
            }
            for (name, t) in pkg.types.iter() {
                let decl = t.content.as_str();
                // split content, 1024B for each
                let mut start = 0;
                let mut end = 1024;
                while start < decl.len() {
                    if end > decl.len() {
                        end = decl.len();
                    }
                    if start >= 1024 {
                        start -= 100;
                    }
                    w.write_record(&[&format!("{}.{}", pname, name), "Type", &decl[start..end]])
                        .unwrap();
                    start = end;
                    end += 924;
                }
            }
            for (name, v) in pkg.vars.iter() {
                let decl = v.content.as_str();
                // split content, 1024B for each
                let mut start = 0;
                let mut end = 1024;
                while start < decl.len() {
                    if end > decl.len() {
                        end = decl.len();
                    }
                    if start >= 1024 {
                        start -= 100;
                    }
                    w.write_record(&[&format!("{}.{}", pname, name), "Var", &decl[start..end]])
                        .unwrap();
                    start = end;
                    end += 924;
                }
            }
        }
    }
    w.flush().unwrap();
    String::from_utf8(w.into_inner().unwrap()).unwrap()
}

pub fn to_csv_pkgs(repo: &Repository) -> String {
    let mut w = Writer::from_writer(Vec::new());
    // add header
    w.write_record(&["Name", "Summary"]).unwrap();

    for (mod_name, _mod) in repo.modules.iter() {
        for (pname, pkg) in _mod.packages.iter() {
            // split comress_data into chunks
            let empty = &"".to_string();
            let sums = split::split_text(pkg.compress_data.as_ref().unwrap_or(empty), 924);
            for sum in sums {
                w.write_record(&[&format!("{}", pname), &format!("{}: {}", pname, sum)])
                    .unwrap();
            }
        }
    }
    w.flush().unwrap();
    String::from_utf8(w.into_inner().unwrap()).unwrap()
}

pub fn export_repo(repo: &Repository) {
    let dir = Path::new(CONFIG.repo_dir.as_str());
    // write summary to csv
    let csv_sum = to_csv_summary(repo);
    let path_sum = dir.join(repo.id.replace("/", "_").to_string() + "_summary.csv");
    let mut file = File::create(&path_sum).unwrap();
    file.write_all(csv_sum.as_bytes()).unwrap();

    // write summary to csv
    let csv_decl = to_csv_decl(repo);
    let path_decl = dir.join(repo.id.replace("/", "_").to_string() + "_decl.csv");
    let mut file = File::create(&path_decl).unwrap();
    file.write_all(csv_decl.as_bytes()).unwrap();

    // write package to csv
    let csv_pkg = to_csv_pkgs(repo);
    let path_pkg = dir.join(repo.id.replace("/", "_").to_string() + "_pkg.csv");
    let mut file = File::create(&path_pkg).unwrap();
    file.write_all(csv_pkg.as_bytes()).unwrap();
}

pub fn force_parse_repo(repo_path: &String, opts: &CompressOptions) -> Result<Repository, Error> {
    let (path, name) = parse_repo_path(repo_path)?;
    // force to parse the repo
    let data = parse_repo(name, &path, opts)?;
    match compress::from_json(&name, String::from_utf8(data).unwrap().as_str()) {
        Ok(repo) => Ok(repo),
        Err(err) => Err(Error::Parse(err.to_string())),
    }
}

fn parse_repo_path(repo_path: &String) -> Result<(PathBuf, &str), Error> {
    let git_dir = Path::new(CONFIG.repo_dir.as_str());

    let ps: Vec<&str> = repo_path.split('/').collect();
    let (path, name) = if repo_path.ends_with(".git") || repo_path.starts_with("https://") {
        // url
        let repo_name = ps[ps.len() - 1].strip_suffix(".git").unwrap();
        let path = git_dir.join(repo_name);
        if !path.exists() {
            // git clone
            git::git_clone(&repo_path, &path).expect("Failed to clone repo");
        }
        (path, repo_name)
    } else {
        // existing path
        let path = git_dir.join(&repo_path);
        if !path.exists() {
            println!("path not exists: {:?}", path);
            return Err(Error::GitCloneError("path not exists".to_string()));
        }
        // directly use the repo name
        (path, ps[0])
    };
    Ok((path, name))
}

pub struct CompressOptions {
    pub export_compress: bool,
    pub not_load_external_symbol: bool,
    pub no_need_comment: bool,
}

pub fn get_repo(repo_path: &String, opts: &CompressOptions) -> Result<Repository, Error> {
    let (path, name) = parse_repo_path(repo_path)?;

    // check if cache the result
    let data = if let Some(data) = cache::get_cache().get(&name) {
        data
    } else {
        // parse the repo
        parse_repo(name, &path, opts)?
    };

    match compress::from_json(&name, String::from_utf8(data).unwrap().as_str()) {
        Ok(repo) => Ok(repo),
        Err(err) => Err(Error::Parse(err.to_string())),
    }
}

fn parse_repo(name: &str, path: &Path, opts: &CompressOptions) -> Result<Vec<u8>, Error> {
    let (parser, args) = config::parser_and_args(path.to_str().unwrap(), opts);
    // parse the repo by parse
    match cmd::run_command_bytes(&parser, args) {
        Ok(output) => {
            cache::get_cache().put(&name, output.clone()).unwrap();
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
