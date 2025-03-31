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
use std::path::{Path, PathBuf};

use lazy_static::lazy_static;
use serde::Deserialize;

use crate::parse;

#[derive(Debug)]
pub enum Language {
    English,
    Chinese,
}

#[derive(Debug)]
pub struct Config {
    pub work_dir: String,
    pub parser_dir: String,
    pub api_type: String,
    pub maas_model_name: String,
    pub mass_http_url: String,

    pub coze_api_token: Option<String>,
    pub coze_bot_id: Option<String>,

    pub ollama_model: Option<String>,

    pub language: Language,
    pub exclude_dirs: Vec<String>,
}

fn default_work_dir() -> String {
    "tmp_abcoder".to_string()
}

fn default_parser_dir() -> String {
    "tools/parser".to_string()
}

fn default_api_type() -> String {
    "maas".to_string()
}

fn default_maas_model_name() -> String {
    "".to_string()
}

impl Config {
    pub fn new() -> Self {
        Self {
            work_dir: default_work_dir(),
            parser_dir: default_parser_dir(),
            api_type: default_api_type(),
            maas_model_name: default_maas_model_name(),
            mass_http_url: "".to_string(),
            coze_api_token: None,
            coze_bot_id: None,
            ollama_model: None,
            exclude_dirs: vec![],
            language: Language::Chinese,
        }
    }

    pub fn parse_from_env() -> Self {
        let s = Self {
            work_dir: std::env::var("WORK_DIR").unwrap_or_else(|_| default_work_dir()),
            parser_dir: std::env::var("PARSER_DIR").unwrap_or_else(|_| default_parser_dir()),
            api_type: std::env::var("API_TYPE").unwrap_or_else(|_| default_api_type()),
            maas_model_name: std::env::var("MAAS_MODEL_NAME")
                .unwrap_or_else(|_| default_maas_model_name()),
            mass_http_url: std::env::var("MASS_HTTP_URL").unwrap_or_else(|_| "".to_string()),
            coze_api_token: std::env::var("COZE_API_TOKEN").ok(),
            coze_bot_id: std::env::var("COZE_BOT_ID").ok(),
            ollama_model: std::env::var("OLLAMA_MODEL").ok(),
            exclude_dirs: std::env::var("EXCLUDE_DIRS")
                .map(|v| v.split(',').map(|s| s.to_string()).collect())
                .unwrap_or_else(|_| vec![]),
            language: std::env::var("LANGUAGE")
                .map(|v| match v.as_str() {
                    "en" => Language::English,
                    "zh" => Language::Chinese,
                    _ => Language::Chinese,
                })
                .unwrap_or(Language::Chinese),
        };
        s
    }
}

lazy_static! {
    pub static ref CONFIG: Config = {
        dotenv::dotenv().ok();
        Config::parse_from_env()
    };
}

pub fn parser_path() -> String {
    Path::new(&CONFIG.parser_dir)
        .join("lang")
        .to_str()
        .unwrap()
        .to_string()
}

pub enum ProgramLanguage {
    Rust,
    Go,
    Unknown(String),
}

impl ProgramLanguage {
    pub fn to_string(&self) -> String {
        match self {
            ProgramLanguage::Rust => "rust".to_string(),
            ProgramLanguage::Go => "go".to_string(),
            ProgramLanguage::Unknown(s) => s.to_string(),
        }
    }
}

fn decide_language(path: &str) -> ProgramLanguage {
    // scan root directory
    walkdir::WalkDir::new(path)
        .max_depth(2)
        .into_iter()
        .filter_map(|entry| {
            let binding = entry.unwrap();
            let path = binding.path();
            if !path.is_dir() {
                let name = path.file_name().unwrap().to_str().unwrap();
                if name == "Cargo.toml" {
                    return Some(ProgramLanguage::Rust);
                }
                if name == "go.mod" {
                    return Some(ProgramLanguage::Go);
                }
            }
            None
        })
        .next()
        .unwrap_or(ProgramLanguage::Unknown(path.to_string()))
}

pub fn parser_and_args<'a>(
    repo_path: &'a str,
    opts: &parse::CompressOptions,
) -> (String, Vec<String>) {
    let lang = decide_language(repo_path);
    let path = parser_path();
    println!("parser path: {:?}", path);
    let mut args = vec![
        "collect".to_string(),
        lang.to_string(),
        repo_path.to_string(),
    ];
    for exclude in &CONFIG.exclude_dirs {
        args.push(format!("--exclude={exclude}"));
    }
    if !opts.not_load_external_symbol {
        args.push("--load-external-symbol".to_string());
    }
    if opts.no_need_comment {
        args.push("--no-need-comment".to_string());
    }
    (path, args)
}
