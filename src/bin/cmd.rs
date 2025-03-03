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

use std::{env, panic, process, thread, time::Duration};

use ABCoder::{
    compress::{compress::compress_all, types::types::CodeCache},
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
}

#[derive(Clone, Debug, Default)]
struct CompressAction {
    export_compress: bool,
    force_update_ast: bool,
    not_load_external_symbol: bool,
}

fn main() {
    // parse Options from cmd args
    let options = parse_options();
    println!("arguments: {:?}", options);

    match options.action {
        Action::Compress(cmp) => {
            if cmp.force_update_ast {
                merge_compress(&options.repo_path, &cmp);
            }
            compress(&options.repo_path, &cmp);
            if cmp.export_compress {
                export_compress(&options.repo_path, &cmp);
            }
        }
    }
}

const USAGE: &str = "Usage: ABCoder <Action> <RepoPath> [Flags]
Action: compress
    compress: compress the repo. Including flags:
      --export-compress: export the compress result
      --force-update-ast: force parsing repo and merge the previous result 
      --not-load-external-symbol: not load external external symbols to speed up parsing";

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
                        "--not-load-external-symbol" => {
                            compress_action.not_load_external_symbol = true;
                        }
                        _ => {}
                    }
                }
            }

            Action::Compress(compress_action)
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

fn compress(repo_path: &String, cmp: &CompressAction) {
    // recoverable logic
    let run = || {
        // get the repo
        let repo = repo::get_repo(repo_path, !cmp.not_load_external_symbol);
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

fn export_compress(repo_path: &String, cmp: &CompressAction) {
    // get the repo
    let repo = repo::get_repo(repo_path, !cmp.not_load_external_symbol);
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

fn merge_compress(repo_path: &String, opts: &CompressAction) {
    // get old repo
    let repo = repo::get_repo(repo_path, !opts.not_load_external_symbol);
    if let Err(err) = repo {
        println!("get repo error: {:?}", err);
        process::exit(1);
    }
    let mut repo = repo.unwrap();

    // parse new repo
    let nrepo = repo::force_parse_repo(repo_path, !opts.not_load_external_symbol);
    if let Err(err) = nrepo {
        println!("parse repo error: {:?}", err);
        process::exit(1);
    }
    let nrepo = nrepo.unwrap();

    repo.merge_with(&nrepo);
    repo.save_to_cache();

    println!("successfully merge repo: {}", repo.id);
}
