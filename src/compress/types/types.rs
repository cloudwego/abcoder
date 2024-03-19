use csv::Writer;
use serde_json::{json, Value};
use std::{clone, collections::HashMap};

use serde::{Deserialize, Serialize};

use crate::{
    compress::llm::split::{self, split_text},
    storage::cache::get_cache,
};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Repository {
    #[serde(default)]
    pub id: String,
    #[serde(rename = "Modules")]
    pub modules: HashMap<String, Module>,
}

impl Repository {
    pub fn contains(&self, id: &Identity) -> bool {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(p) = m.packages.get(&id.pkg_path) {
                if let Some(_) = p.functions.get(&id.name) {
                    return true;
                } else if let Some(_) = p.types.get(&id.name) {
                    return true;
                }
            }
        }
        false
    }

    pub fn merge_with(&mut self, other: &Repository) {
        for (mod_name, _mod) in other.modules.iter() {
            if let Some(smod) = self.modules.get_mut(mod_name) {
                for (pkg_name, pkg) in _mod.packages.iter() {
                    if let Some(p) = smod.packages.get_mut(pkg_name) {
                        p.merge_with(pkg);
                    } else {
                        smod.packages.insert(pkg_name.clone(), pkg.clone());
                    }
                    smod.name = _mod.name.clone();
                    smod.dir = _mod.dir.clone();
                    smod.dependencies = _mod.dependencies.clone();
                }
            } else {
                self.modules.insert(mod_name.clone(), _mod.clone());
            }
        }
    }

    pub fn save_to_cache(&self) {
        let repo = serde_json::to_string(&self).expect("marshal struct error");
        get_cache()
            .put(self.id.as_ref(), Vec::from(repo))
            .expect("save to cache failed");
    }

    pub fn get_func(&self, id: &Identity) -> Option<&Function> {
        if let Some(m) = self.modules.get(&id.name) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.functions.get(&id.name) {
                    return Some(f);
                }
            }
        }
        None
    }

    pub fn get_type(&self, id: &Identity) -> Option<&Struct> {
        if let Some(m) = self.modules.get(&id.name) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.types.get(&id.name) {
                    return Some(f);
                }
            }
        }
        None
    }

    pub fn to_csv_summary(&self) -> String {
        let mut w = Writer::from_writer(Vec::new());
        // add header
        w.write_record(&["Package", "Name", "Kind", "Signature", "Summary"])
            .unwrap();

        for (mod_name, _mod) in self.modules.iter() {
            for (pname, pkg) in _mod.packages.iter() {
                for (name, f) in pkg.functions.iter() {
                    let tmp = &"".to_string();
                    // split content, 1024B for each
                    let sums = split_text(f.compress_data.as_ref().unwrap_or(tmp), 924);
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
                    let sums = split_text(t.compress_data.as_ref().unwrap_or(tmp), 924);
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
            }
        }
        w.flush().unwrap();
        String::from_utf8(w.into_inner().unwrap()).unwrap()
    }

    pub fn to_csv_decl(&self) -> String {
        let mut w = Writer::from_writer(Vec::new());
        // add header
        w.write_record(&["Identity", "Kind", "Definition"]).unwrap();

        for (mod_name, _mod) in self.modules.iter() {
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
                        w.write_record(&[
                            &format!("{}.{}", pname, name),
                            "Type",
                            &decl[start..end],
                        ])
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

    pub fn to_csv_pkgs(&self) -> String {
        let mut w = Writer::from_writer(Vec::new());
        // add header
        w.write_record(&["Name", "Summary"]).unwrap();

        for (mod_name, _mod) in self.modules.iter() {
            for (pname, pkg) in _mod.packages.iter() {
                // split comress_data into chunks
                let empty = &"".to_string();
                let sums = split_text(pkg.compress_data.as_ref().unwrap_or(empty), 924);
                for sum in sums {
                    w.write_record(&[&format!("{}", pname), &format!("{}: {}", pname, sum)])
                        .unwrap();
                }
            }
        }
        w.flush().unwrap();
        String::from_utf8(w.into_inner().unwrap()).unwrap()
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Module {
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Dir")]
    pub dir: String,
    #[serde(rename = "Dependencies")]
    pub dependencies: HashMap<String, String>,
    #[serde(rename = "Packages")]
    pub packages: HashMap<String, Package>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Package {
    #[serde(rename = "PkgPath", default)]
    pub id: String,
    // #[serde(rename = "Dependencies", default)]
    // pub dependencies: Vec<String>,
    #[serde(rename = "Functions")]
    pub functions: HashMap<String, Function>,
    #[serde(rename = "Types")]
    pub types: HashMap<String, Struct>,

    pub compress_data: Option<String>,
}

impl Package {
    pub fn merge_with(&mut self, other: &Package) {
        self.id = other.id.clone();
        for (name, f) in other.functions.iter() {
            if let Some(func) = self.functions.get_mut(name) {
                func.merge_with(f);
            } else {
                self.functions.insert(name.clone(), f.clone());
            }
        }
        for (name, t) in other.types.iter() {
            if let Some(typ) = self.types.get_mut(name) {
                typ.merge_with(t);
            } else {
                self.types.insert(name.clone(), t.clone());
            }
        }
    }

    // output a json string of exported functions and types, schema:
    // {"Functions":[{"Name":"", "Description":"", "Signature":""}], "Types":[{"Name":"", "Description":"", "Signature":""}]}
    pub fn export_api(&self) -> String {
        // functions
        let mut funcs = Vec::new();
        for (name, f) in self.functions.iter() {
            let sig: &str = f.content.split_once('\n').unwrap_or((&f.content, "")).0;
            // skip non-exported functions
            if !f.is_exported {
                continue;
            }
            funcs.push(json!({
                "Name": name,
                "Description": f.compress_data,
                "Signature": sig,
            }));
        }
        // types
        let mut types = Vec::new();
        for (name, t) in self.types.iter() {
            let sig = t.content.split_once('\n').unwrap_or((&t.content, "")).0;
            // skip non-exported types
            if !t.is_exported {
                continue;
            }
            types.push(json!({
                "Name": name,
                "Description": t.compress_data,
                "Signature": sig,
            }));
        }

        json!({
            "PkgPath": self.id,
            "Functions": funcs,
            "Types": types
        })
        .to_string()
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Function {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "IsMethod")]
    is_method: bool,
    #[serde(rename = "Content")]
    pub content: String,
    #[serde(rename = "AssociatedStruct")]
    associated_struct: Option<Identity>,
    #[serde(rename = "FunctionCalls")]
    pub function_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "MethodCalls")]
    pub method_calls: Option<HashMap<String, Identity>>,

    // compress_data
    pub compress_data: Option<String>,
}

impl Function {
    pub fn id(&self) -> Identity {
        Identity {
            mod_path: self.mod_path.clone(),
            pkg_path: self.pkg_path.clone(),
            name: self.name.clone(),
        }
    }

    pub fn merge_with(&mut self, other: &Function) {
        let compress = self.compress_data.clone();
        *self = other.clone();
        self.compress_data = compress;
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Struct {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub(crate) pkg_path: String,
    #[serde(rename = "Name")]
    pub(crate) name: String,
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "TypeKind")]
    type_kind: u8,
    #[serde(rename = "Content")]
    pub(crate) content: String,
    #[serde(rename = "SubStruct")]
    pub(crate) sub_struct: Option<HashMap<String, Identity>>,
    #[serde(rename = "InlineStruct")]
    pub(crate) inline_struct: Option<HashMap<String, Identity>>,
    #[serde(rename = "Methods")]
    pub(crate) methods: Option<HashMap<String, Identity>>,

    // compress_data
    pub compress_data: Option<String>,
}

impl Struct {
    pub fn id(&self) -> Identity {
        Identity {
            mod_path: self.mod_path.clone(),
            pkg_path: self.pkg_path.clone(),
            name: self.name.clone(),
        }
    }
    pub fn merge_with(&mut self, other: &Struct) {
        let compress = self.compress_data.clone();
        *self = other.clone();
        self.compress_data = compress;
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Identity {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct ToCompressFunc {
    #[serde(rename = "Content")]
    pub(crate) content: String,
    #[serde(rename = "Related_func")]
    pub(crate) related_func: Option<Vec<CalledType>>,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct ToCompressType {
    #[serde(rename = "Content")]
    pub(crate) content: String,
    #[serde(rename = "Related_methods")]
    pub(crate) related_methods: Option<Vec<KeyValueType>>,
    #[serde(rename = "Related_types")]
    pub(crate) related_types: Option<Vec<KeyValueType>>,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct CalledType {
    #[serde(rename = "CallName")]
    pub call_name: String,
    #[serde(rename = "Description")]
    pub description: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct KeyValueType {
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Description")]
    pub description: String,
}
