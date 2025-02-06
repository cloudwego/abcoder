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

use std::{env, fs, os, panic, path::Path, process, thread, time::Duration};

use ABCoder::{
    compress::{
        compress::{compress_all, from_json},
        conv::convert_go2rust,
        types::types::CodeCache,
    },
    config::CONFIG,
    repo,
};

#[derive(Clone, Debug)]
struct Options {
    repo_path: String,
    action: Action,
}

#[derive(Clone, Debug)]
enum Action {
    Compress(CompressAction),
    Translate(TranslateAction),
}

#[derive(Clone, Debug, Default)]
struct CompressAction {
    export_compress: bool,
    force_update_ast: bool,
}

#[derive(Clone, Debug)]
enum TranslateAction {
    Go2Rust,
    Unknown,
}

fn main() {
    // parse Options from cmd args
    let options = parse_options();
    println!("arguments: {:?}", options);

    match options.action {
        Action::Compress(cmp) => {
            if cmp.force_update_ast {
                merge_compress(&options.repo_path);
            }
            compress(&options.repo_path);
            if cmp.export_compress {
                export_compress(&options.repo_path);
            }
        }
        Action::Translate(trans) => {
            match trans {
                TranslateAction::Go2Rust => {
                    // go转rust
                    trans_go2rust(&options.repo_path);
                }
                TranslateAction::Unknown => {
                    println!("{}", USAGE);
                    process::exit(1);
                }
            }
        }
    }
}

const USAGE: &str = "Usage: ABCoder <Action> <RepoPath> [Flags]
Action: compress | translate
    compress: compress the repo. Including flags:
      --export-compress: export the compress result
      --force-update-ast: force parsing repo and merge the previous result 
    translate: translate the repo. Including flags:
      --go2rust: translate the repo from go to rust
";

fn parse_options() -> Options {
    let args: Vec<String> = env::args().collect();
    if args.len() < 3 {
        println!("{}", USAGE);
        process::exit(1);
    }

    let action = match args[1].as_str() {
        "compress" => {
            let mut compress_action = CompressAction::default();
            if args.len() > 3 {
                for i in 3..args.len() {
                    match args[i].as_str() {
                        "--export-compress" => {
                            compress_action.export_compress = true;
                        }
                        "--force-update-ast" => {
                            compress_action.force_update_ast = true;
                        }
                        _ => {}
                    }
                }
            }

            Action::Compress(compress_action)
        }
        "translate" => {
            let mut translate_action = TranslateAction::Unknown;
            if args.len() > 3 {
                for i in 3..args.len() {
                    match args[i].as_str() {
                        "--go2rust" => {
                            translate_action = TranslateAction::Go2Rust;
                        }
                        _ => {}
                    }
                }
            }
            Action::Translate(translate_action)
        }
        _ => {
            println!("{}", USAGE);
            process::exit(1);
        }
    };

    Options {
        repo_path: args[2].clone(),
        action,
    }
}

fn compress(repo_path: &String) {
    // recoverable logic
    let run = || {
        // get the repo
        let repo = repo::get_repo(repo_path);
        if let Err(err) = repo {
            println!("get repo error: {:?}", err);
            process::exit(1);
        }

        let mut repo = repo.unwrap();

        // compress the repo
        println!("compressing repo: {}", repo.id);
        // block on compress
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(compress_all(&mut repo));

        // save the compressed repo
        repo.save_to_cache();

        println!("successfully compressed repo: {}", repo.id);
    };

    loop {
        let result = panic::catch_unwind(run);
        if let Err(err) = result {
            println!("panic: {:?}", err);
            // sleep 60s and retry
            thread::sleep(Duration::from_secs(60));
            continue;
        } else {
            process::exit(0);
        }
    }
}

fn export_compress(repo_path: &String) {
    // get the repo
    let repo = repo::get_repo(repo_path);
    if let Err(err) = repo {
        println!("get repo error: {:?}", err);
        process::exit(1);
    }

    let mut repo = repo.unwrap();

    // export the compress
    println!("export repo: {}", repo.id);
    repo::export_repo(&mut repo);
    // save the compressed repo
    repo.save_to_cache();

    println!("successfully compressed repo: {}", repo.id);
}

fn merge_compress(repo_path: &String) {
    // get old repo
    let repo = repo::get_repo(repo_path);
    if let Err(err) = repo {
        println!("get repo error: {:?}", err);
        process::exit(1);
    }
    let mut repo = repo.unwrap();

    // parse new repo
    let nrepo = repo::force_parse_repo(repo_path);
    if let Err(err) = nrepo {
        println!("parse repo error: {:?}", err);
        process::exit(1);
    }
    let nrepo = nrepo.unwrap();

    repo.merge_with(&nrepo);
    repo.save_to_cache();

    println!("successfully merge repo: {}", repo.id);
}

fn trans_go2rust(repo_path: &String) {
    let run = || {
        // get the repo
        let repo = repo::get_repo(repo_path);
        if let Err(err) = repo {
            println!("get repo error: {:?}", err);
            process::exit(1);
        }

        let mut repo = repo.unwrap();

        // export the compress
        println!("export repo: {}", repo.id);

        let mut m = CodeCache::new(format!("go2rust_{}", &repo.id));
        m.load_from_cache();
        let out_dir = Path::new(&CONFIG.repo_dir).join("go2rust").join(&repo.id);

        // block on convert_go2rust
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(convert_go2rust(
            &mut repo,
            out_dir.to_str().unwrap(),
            &mut m,
        ));

        // save the compressed repo
        repo.save_to_cache();

        println!("successfully compressed repo: {}", repo.id);
    };

    loop {
        let result = panic::catch_unwind(run);
        if let Err(err) = result {
            println!("panic: {:?}", err);
            // sleep 60s and retry
            thread::sleep(Duration::from_secs(60));
            continue;
        } else {
            process::exit(0);
        }
    }
}
