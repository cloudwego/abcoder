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

#[derive(Clone, Debug, Default)]
pub struct ExportOptions {
    pub csv: bool,
    pub public_only: bool,
    pub output: Option<String>,
}

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

pub fn to_markdown(repo: &Repository, opts: &ExportOptions) -> String {
    let mut md = String::new();

    for (mod_name, module) in repo.modules.iter() {
        if repo.is_external_mod(mod_name) {
            continue;
        }

        // 添加模块标题
        md.push_str(&format!("# {}\n\n", mod_name));
        let lang = &module.language;

        for (pkg_name, pkg) in module.packages.iter() {
            // 添加包标题
            md.push_str(&format!("## {}\n\n", pkg_name));
            if let Some(data) = &pkg.compress_data {
                md.push_str(&format!("{}\n\n", data));
            }

            // 添加函数
            for (func_name, func) in pkg.functions.iter() {
                if opts.public_only && !repo.is_exported(&func.id()) {
                    continue;
                }

                md.push_str(&format!("### {}\n\n", func_name));
                if let Some(data) = &func.compress_data {
                    md.push_str(&format!("{}\n\n", data));
                }
                md.push_str(&format!("- Position\n\n{}:{}\n\n", func.file, func.line));
                md.push_str(&format!(
                    "- Codes\n\n```{}\n{}\n```\n\n",
                    lang, func.content
                ));
            }

            // 添加类型
            for (type_name, typ) in pkg.types.iter() {
                if opts.public_only && !repo.is_exported(&typ.id()) {
                    continue;
                }

                md.push_str(&format!("### {}\n\n", type_name));
                if let Some(data) = &typ.compress_data {
                    md.push_str(&format!("{}\n\n", data));
                }
                md.push_str(&format!("- Position\n\n{}:{}\n\n", typ.file, typ.line));
                md.push_str(&format!("- Codes\n\n```{}\n{}\n```\n\n", lang, typ.content));
            }

            // 添加变量
            for (var_name, var) in pkg.vars.iter() {
                if opts.public_only && !repo.is_exported(&var.id()) {
                    continue;
                }

                md.push_str(&format!("### {}\n\n", var_name));
                if let Some(data) = &var.compress_data {
                    md.push_str(&format!("{}\n\n", data));
                }
                md.push_str(&format!("- Position\n\n{}:{}\n\n", var.file, var.line));
                md.push_str(&format!("- Codes\n\n```{}\n{}\n```\n\n", lang, var.content));
            }
        }
    }

    md
}

// ... existing code ...

pub fn export_repo(repo: &Repository, opts: &ExportOptions) {
    let dir = if let Some(path) = &opts.output {
        Path::new(path)
    } else {
        // pwd
        Path::new(&CONFIG.work_dir)
    };

    if opts.csv {
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
    } else {
        // write markdown
        let md = to_markdown(repo, opts);
        let path_md = dir.join(repo.id.replace("/", "_").to_string() + ".md");
        let mut file = File::create(&path_md).unwrap();
        file.write_all(md.as_bytes()).unwrap();
    }
}
