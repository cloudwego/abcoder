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
    compress::compress::compress_all,
    export::{self, ExportOptions},
    parse::{self, CompressOptions},
};

#[derive(Clone, Debug)]
struct Options {
    repo_path: String,
    action: Action,
}

#[derive(Clone, Debug)]
enum Action {
    Compress(CompressOptions),
    Export(ExportOptions),
}

fn main() {
    // parse Options from cmd args
    let options = parse_options();
    println!("arguments: {:?}", options);

    match options.action {
        Action::Compress(cmp) => {
            if cmp.force_update_ast {
                merge_repo(&options.repo_path, &cmp);
            }
            compress(&options.repo_path, &cmp);
        }
        Action::Export(exp) => {
            export(&options.repo_path, &exp);
        }
    }
}

const USAGE: &str = "Usage: ABCoder <Action> <RepoPath> [Flags]
RepoPath: the path of the repo to compress. Can be a local path or a git url.
Actions: compress|export
compress: compress the repo. Including flags:
    --parse-only: only parse the repo, not compress it
    --export-compress: export the compress result
    --force-update-ast: force parsing repo and merge the previous result 
    --not-load-external-symbol: not load external external symbols to speed up parsing
    --no-need-comment: not need comment in symbol content (only works for Go now)
export: export the compress result to csv or markdown (default). Including flags:
    --csv: export the compress result to csv
    --out-dir <path>: output directory path, default is $WORK_DIR
    --public-only: only export the public symbols
";

fn parse_options() -> Options {
    let args: Vec<String> = env::args().collect();
    if args.len() < 3 {
        println!("{}", USAGE);
        process::exit(1);
    }

    let action = match args[1].as_str() {
        "compress" => {
            let mut compress_action = CompressOptions::default();
            if args.len() > 3 {
                for i in 3..args.len() {
                    match args[i].as_str() {
                        "--force-update-ast" => {
                            compress_action.force_update_ast = true;
                        }
                        "--not-load-external-symbol" => {
                            compress_action.not_load_external_symbol = true;
                        }
                        "--no-need-comment" => {
                            compress_action.no_need_comment = true;
                        }
                        _ => {}
                    }
                }
            }
            Action::Compress(compress_action)
        }

        "export" => {
            let mut opts = ExportOptions::default();
            if args.len() > 3 {
                for i in 3..args.len() {
                    match args[i].as_str() {
                        "--out-dir" => {
                            if args.len() <= i + 1 {
                                println!("--out-dir must specify a value");
                                process::exit(1);
                            }
                            opts.output = Some(args[i + 1].clone());
                        }
                        "--csv" => {
                            opts.csv = true;
                        }
                        "--public-only" => {
                            opts.public_only = true;
                        }
                        _ => {}
                    }
                }
            }
            Action::Export(opts)
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

fn compress(repo_path: &String, cmp: &CompressOptions) {
    // recoverable logic
    let run = || {
        // get the repo
        let repo = parse::get_repo(repo_path, &cmp);
        if let Err(err) = repo {
            println!("get repo error: {:?}", err);
            process::exit(1);
        }

        let mut repo = repo.unwrap();
        repo.save_to_cache();
        println!("successfully parsed repo: {}", repo.id);
        if cmp.parse_only {
            return;
        }

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

fn export(repo_path: &String, cmp: &ExportOptions) {
    // get the repo
    let repo = parse::get_repo(repo_path, &CompressOptions::default());
    if let Err(err) = repo {
        println!("get repo error: {:?}", err);

        process::exit(1);
    }

    let mut repo = repo.unwrap();

    // export the compress
    println!("export repo: {}", repo.id);
    export::export_repo(&mut repo, cmp);

    println!("successfully exported repo: {}", repo.id);
}

fn merge_repo(repo_path: &String, cmp: &CompressOptions) {
    // get old repo
    let repo = parse::get_repo(repo_path, &cmp);
    if let Err(err) = repo {
        println!("get repo error: {:?}", err);
        process::exit(1);
    }
    let mut repo = repo.unwrap();

    // parse new repo
    let nrepo = parse::force_parse_repo(repo_path, &cmp);
    if let Err(err) = nrepo {
        println!("parse repo error: {:?}", err);
        process::exit(1);
    }
    let nrepo = nrepo.unwrap();

    repo.merge_with(&nrepo);
    repo.save_to_cache();

    println!("successfully merge repo: {}", repo.id);
}
