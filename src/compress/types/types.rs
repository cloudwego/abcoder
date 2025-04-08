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

use serde_json::{json, Value};
use std::{
    cell::RefCell,
    collections::{HashMap, HashSet},
    hash::{Hash, Hasher},
    rc::Rc,
};

use crate::storage::cache::get_cache;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Repository {
    #[serde(default)]
    pub id: String,
    #[serde(rename = "Modules")]
    pub modules: HashMap<String, Module>,

    // constructed by ABCoder
    #[serde(rename = "Graph")]
    pub graph: Option<HashMap<String, Node>>,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct Node {
    #[serde(rename = "Type")]
    pub r#type: NodeType,
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub(crate) pkg_path: String,
    #[serde(rename = "Name")]
    pub(crate) name: String,
    #[serde(rename = "Dependencies")]
    pub dependencies: Option<Vec<Relation>>,
    #[serde(rename = "References")]
    pub references: Option<Vec<Relation>>,
}

impl Node {
    pub fn id(&self) -> Identity {
        Identity {
            mod_path: self.mod_path.clone(),
            pkg_path: self.pkg_path.clone(),
            name: self.name.clone(),
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub enum RelationKind {
    Dependency,
    Reference,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Relation {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub(crate) pkg_path: String,
    #[serde(rename = "Name")]
    pub(crate) name: String,
    #[serde(rename = "Kind")]
    pub(crate) kind: RelationKind,
    #[serde(rename = "Desc")]
    pub(crate) desc: String,
}

impl Relation {
    pub fn id(&self) -> Identity {
        Identity {
            mod_path: self.mod_path.clone(),
            pkg_path: self.pkg_path.clone(),
            name: self.name.clone(),
        }
    }
}

#[derive(Serialize, Debug, Clone, Default)]
pub enum NodeType {
    #[default]
    Unknown,
    Func,
    Type,
    Var,
}

impl NodeType {
    pub fn to_string(&self) -> String {
        match self {
            NodeType::Func => "FUNC".to_string(),
            NodeType::Type => "TYPE".to_string(),
            NodeType::Var => "VAR".to_string(),
            _ => "UNKNOWN".to_string(),
        }
    }
}

impl<'de> Deserialize<'de> for NodeType {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        match s.as_str() {
            "FUNC" => Ok(NodeType::Func),
            "TYPE" => Ok(NodeType::Type),
            "VAR" => Ok(NodeType::Var),
            _ => Ok(NodeType::Unknown),
        }
    }
}

impl Repository {
    pub fn is_external_mod(&self, mod_path: &str) -> bool {
        return mod_path.contains("@") || mod_path == "std" || mod_path == "";
    }

    pub fn get_id_content(&self, id: &Identity) -> Option<String> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.functions.get(&id.name) {
                    return Some(f.content.clone());
                } else if let Some(t) = pkg.types.get(&id.name) {
                    return Some(t.content.clone());
                } else if let Some(v) = pkg.vars.get(&id.name) {
                    return Some(v.content.clone());
                }
            }
        }
        None
    }

    pub fn get_id_content_ref(&self, id: &Identity) -> Option<&String> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.functions.get(&id.name) {
                    return Some(&f.content);
                } else if let Some(t) = pkg.types.get(&id.name) {
                    return Some(&t.content);
                } else if let Some(v) = pkg.vars.get(&id.name) {
                    return Some(&v.content);
                }
            }
        }
        None
    }

    pub fn get_pkg(&self, id: &Identity) -> Option<(&Module, &Package)> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                return Some((m, pkg));
            }
        }
        None
    }

    pub fn inside_main_pkg(&self, m: &str, pkg: &str) -> Option<&String> {
        if let Some(m) = self.modules.get(m) {
            for (name, p) in m.packages.iter() {
                if p.is_main && pkg.starts_with(name) {
                    return Some(&p.id);
                }
            }
        }
        None
    }

    pub fn remove_id(&mut self, id: &Identity) {
        for (_, _mod) in self.modules.iter_mut() {
            for (_, pkg) in _mod.packages.iter_mut() {
                if let Some(_) = pkg.functions.remove(&id.name) {
                    continue;
                } else if let Some(_) = pkg.types.remove(&id.name) {
                    continue;
                } else if let Some(_) = pkg.vars.remove(&id.name) {
                    continue;
                }
            }
        }
    }

    pub fn remove_unreferenced(&mut self, reffered: &HashSet<&Identity>) {
        // filter all external (or "kitex_gen" or "hertz_gen") and not referred nodes in repo
        for (_, v) in self.graph.clone().unwrap().iter() {
            if (!v.id().inside()
                || v.pkg_path.contains("kitex_gen")
                || v.pkg_path.contains("hertz_gen"))
                && !reffered.contains(&v.id())
            {
                // remove id in repo
                self.remove_id(&v.id());
            }
        }
    }

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
        self.graph = other.graph.clone();
    }

    pub fn save_to_cache(&self) {
        let repo = serde_json::to_string(&self).expect("marshal struct error");
        get_cache()
            .put(self.id.as_ref(), Vec::from(repo))
            .expect("save to cache failed");
    }

    pub fn get_func(&self, id: &Identity) -> Option<&Function> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.functions.get(&id.name) {
                    return Some(f);
                }
            }
        }
        None
    }

    pub fn get_type(&self, id: &Identity) -> Option<&Struct> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.types.get(&id.name) {
                    return Some(f);
                }
            }
        }
        None
    }

    pub fn get_kind(&self, id: &Identity) -> NodeType {
        if let Some(func) = self.get_func(id) {
            return NodeType::Func;
        } else if let Some(t) = self.get_type(id) {
            return NodeType::Type;
        } else if let Some(v) = self.get_var(id) {
            return NodeType::Var;
        } else if let Some(code) = self.get_id_content(id) {
            NodeType::Unknown
        } else {
            return NodeType::Unknown;
        }
    }

    pub fn get_file_line<'a>(&'a self, id: &'a Identity) -> FileLine {
        if let Some(func) = self.get_func(id) {
            return FileLine {
                pkg: &id.pkg_path,
                file: &func.file,
                line: func.line,
            };
        } else if let Some(t) = self.get_type(id) {
            return FileLine {
                pkg: &id.pkg_path,
                file: &t.file,
                line: t.line,
            };
        } else if let Some(v) = self.get_var(id) {
            return FileLine {
                pkg: &id.pkg_path,
                file: &v.file,
                line: v.line,
            };
        } else {
            FileLine {
                pkg: "",
                file: "",
                line: 0,
            }
        }
    }

    pub fn is_exported(&self, id: &Identity) -> bool {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.functions.get(&id.name) {
                    return f.is_exported;
                } else if let Some(t) = pkg.types.get(&id.name) {
                    return t.is_exported;
                } else if let Some(v) = pkg.vars.get(&id.name) {
                    return v.is_exported;
                }
            }
        }
        false
    }

    pub fn get_var(&self, id: &Identity) -> Option<&Variant> {
        if let Some(m) = self.modules.get(&id.mod_path) {
            if let Some(pkg) = m.packages.get(&id.pkg_path) {
                if let Some(f) = pkg.vars.get(&id.name) {
                    return Some(f);
                }
            }
        }
        None
    }

    pub fn get_node(&self, id: &Identity) -> Option<&Node> {
        if let Some(graph) = &self.graph {
            return graph.get(&String::from(id));
        }
        None
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Module {
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Dir")]
    pub dir: String,
    #[serde(rename = "Dependencies")]
    pub dependencies: Option<HashMap<String, String>>,
    #[serde(rename = "Packages")]
    pub packages: HashMap<String, Package>,
    #[serde(rename = "Files")]
    pub files: Option<HashMap<String, File>>,
    #[serde(rename = "Language", default)]
    pub language: String,
    #[serde(rename = "compress_data")]
    pub compress_data: Option<String>,
}

impl Module {
    pub fn to_compress(&self) -> ToCompressModule {
        let mut packages = Vec::new();
        for (_, p) in self.packages.iter() {
            packages.push(Description {
                name: &p.id,
                description: p.compress_data.as_ref().unwrap(),
            });
        }

        ToCompressModule {
            name: &self.name,
            dir: &self.dir,
            packages: Some(packages),
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct File {
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Path")]
    pub path: String,
    #[serde(rename = "Imports")]
    pub imports: Option<Vec<Import>>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Import {
    #[serde(rename = "Alias")]
    pub alias: Option<String>,
    #[serde(rename = "Path")]
    pub path: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Package {
    #[serde(rename = "PkgPath", default)]
    pub id: String,
    #[serde(rename = "IsMain", default)]
    pub is_main: bool,
    #[serde(rename = "Dependencies", default)]
    pub dependencies: Vec<String>,
    #[serde(rename = "Functions")]
    pub functions: HashMap<String, Function>,
    #[serde(rename = "Types")]
    pub types: HashMap<String, Struct>,
    #[serde(rename = "Vars")]
    pub vars: HashMap<String, Variant>,
    #[serde(rename = "compress_data")]
    pub compress_data: Option<String>,
}

fn format_file(file: &str) -> String {
    // count non-alphanumeric characters
    let mut i = 0;
    for c in file.chars() {
        if !c.is_alphanumeric() {
            i += 1;
        }
    }
    if i >= 2 {
        // replace non-alphanumeric characters with '_' before last
        file.replace(
            |c: char| -> bool {
                if !c.is_alphanumeric() && i > 1 {
                    i -= 1;
                    return true;
                }
                false
            },
            "_",
        )
    } else {
        file.to_string()
    }
}

#[test]
fn test_format_file() {
    assert_eq!(format_file("a/b/c.go"), "a_b_c.go");
    assert_eq!(format_file("x.pb.go"), "x_pb.go");
}

impl Package {
    pub fn to_files(&self) -> HashMap<String, Vec<Identity>> {
        let mut files: HashMap<String, Vec<Identity>> = HashMap::new();
        for (_, f) in self.functions.iter() {
            let file = format_file(&f.file);
            if let Some(nodes) = files.get_mut(&file) {
                nodes.push(f.id());
            } else {
                files.insert(file, vec![f.id()]);
            }
        }
        for (_, t) in self.types.iter() {
            let file = format_file(&t.file);
            if let Some(nodes) = files.get_mut(&file) {
                nodes.push(t.id());
            } else {
                files.insert(file, vec![t.id()]);
            }
        }
        for (_, v) in self.vars.iter() {
            let file = format_file(&v.file);
            if let Some(nodes) = files.get_mut(&file) {
                nodes.push(v.id());
            } else {
                files.insert(file, vec![v.id()]);
            }
        }
        files
    }

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
        for (name, v) in other.vars.iter() {
            if let Some(var) = self.vars.get_mut(name) {
                var.merge_with(v);
            } else {
                self.vars.insert(name.clone(), v.clone());
            }
        }
    }

    // output a json string of exported functions and types, schema:
    // {"Functions":[{"Name":"", "Description":"", "Signature":""}], "Types":[{"Name":"", "Description":"", "Signature":""}]}
    pub fn to_compress(&self) -> ToCompressPkg {
        // functions
        let mut funcs = Vec::new();
        for (name, f) in self.functions.iter() {
            // skip non-exported functions
            if !f.is_exported {
                continue;
            }
            if let Some(d) = &f.compress_data {
                funcs.push(Description {
                    name: name,
                    description: d,
                });
            }
        }
        // types
        let mut types = Vec::new();
        for (name, t) in self.types.iter() {
            // skip non-exported types
            if !t.is_exported {
                continue;
            }
            if let Some(d) = &t.compress_data {
                types.push(Description {
                    name: name,
                    description: d,
                });
            }
        }
        // vars
        let mut vars = Vec::new();
        for (name, t) in self.vars.iter() {
            // skip non-exported types
            if !t.is_exported {
                continue;
            }
            if let Some(d) = &t.compress_data {
                vars.push(Description {
                    name: name,
                    description: d,
                });
            }
        }
        return ToCompressPkg {
            path: &self.id,
            funcs: Some(funcs),
            types: Some(types),
            vars: Some(vars),
        };
    }
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct Variant {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "File")]
    pub file: String,
    #[serde(rename = "Line")]
    pub line: u32,
    #[serde(rename = "IsExported", default)]
    pub is_exported: bool,
    #[serde(rename = "IsConst", default)]
    pub is_const: bool,
    #[serde(rename = "Type")]
    pub type_id: Option<Identity>,
    #[serde(rename = "IsPointer", default)]
    pub is_pointer: bool,
    #[serde(rename = "Content")]
    pub content: String,

    // compress_data
    #[serde(rename = "compress_data")]
    pub compress_data: Option<String>,
}

impl Variant {
    pub fn id(&self) -> Identity {
        Identity {
            mod_path: self.mod_path.clone(),
            pkg_path: self.pkg_path.clone(),
            name: self.name.clone(),
        }
    }

    pub fn merge_with(&mut self, other: &Variant) {
        let compress = self.compress_data.clone();
        *self = other.clone();
        self.compress_data = compress;
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
    #[serde(rename = "File")]
    pub file: String,
    #[serde(rename = "Line")]
    pub line: u32,
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "IsMethod", default)]
    is_method: bool,
    #[serde(rename = "IsInterfaceMethod", default)]
    is_interface_method: bool,
    #[serde(rename = "Content")]
    pub content: String,
    #[serde(rename = "Receiver")]
    pub receiver: Option<Receiver>,
    #[serde(rename = "Params")]
    pub params: Option<Vec<Identity>>,
    #[serde(rename = "Results")]
    pub results: Option<Vec<Identity>>,
    #[serde(rename = "FunctionCalls")]
    pub function_calls: Option<Vec<Identity>>,
    #[serde(rename = "MethodCalls")]
    pub method_calls: Option<Vec<Identity>>,
    #[serde(rename = "Types")]
    pub types: Option<Vec<Identity>>,
    #[serde(rename = "GlobalVars")]
    pub global_vars: Option<Vec<Identity>>,

    // compress_data
    #[serde(rename = "compress_data")]
    pub compress_data: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Receiver {
    #[serde(rename = "IsPointer")]
    pub is_pointer: bool,
    #[serde(rename = "Type")]
    pub type_id: Identity,
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
    #[serde(rename = "File")]
    pub file: String,
    #[serde(rename = "Line")]
    pub line: u32,
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "TypeKind")]
    pub type_kind: String,
    #[serde(rename = "Content")]
    pub(crate) content: String,
    #[serde(rename = "SubStruct")]
    pub(crate) sub_struct: Option<Vec<Identity>>,
    #[serde(rename = "InlineStruct")]
    pub(crate) inline_struct: Option<Vec<Identity>>,
    #[serde(rename = "Methods")]
    pub(crate) methods: Option<HashMap<String, Identity>>,

    // compress_data
    #[serde(rename = "compress_data")]
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

#[derive(Serialize, Deserialize, Debug, Clone, Eq, Default)]
pub struct Identity {
    #[serde(rename = "ModPath")]
    pub mod_path: String,
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
    // #[serde(rename = "Extra", skip_serializing_if = "Option::is_none")]
    // pub extra: Option<HashMap<String, Value>>,
}

impl Identity {
    pub fn inside(&self) -> bool {
        self.mod_path != "" && !self.mod_path.contains("@")
    }

    pub fn to_string(&self) -> String {
        format!("{}?{}#{}", self.mod_path, self.pkg_path, self.name,)
    }
}

impl From<&Identity> for String {
    fn from(id: &Identity) -> Self {
        format!("{}?{}#{}", id.mod_path, id.pkg_path, id.name,)
    }
}

impl From<&String> for Identity {
    fn from(s: &String) -> Self {
        let mut parts = s.split('?');
        let mod_path = parts.next().unwrap();
        let mut parts = parts.next().unwrap().split('#');
        let pkg_path = parts.next().unwrap();
        let name = parts.next().unwrap();
        Identity {
            mod_path: mod_path.to_string(),
            pkg_path: pkg_path.to_string(),
            name: name.to_string(),
        }
    }
}

impl Hash for Identity {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.mod_path.hash(state);
        self.pkg_path.hash(state);
        self.name.hash(state);
    }
}

impl PartialEq for Identity {
    fn eq(&self, other: &Self) -> bool {
        self.mod_path == other.mod_path
            && self.pkg_path == other.pkg_path
            && self.name == other.name
    }
}

#[derive(PartialEq, Eq, PartialOrd, Ord)]
pub struct FileLine<'a> {
    pub pkg: &'a str,
    pub file: &'a str,
    pub line: u32,
}

#[derive(Serialize, Debug)]
pub(crate) struct ToCompressPkg<'a> {
    #[serde(rename = "PkgPath")]
    pub(crate) path: &'a str,
    #[serde(rename = "Functions")]
    pub(crate) funcs: Option<Vec<Description<'a>>>,
    #[serde(rename = "Types")]
    pub(crate) types: Option<Vec<Description<'a>>>,
    #[serde(rename = "Variables")]
    pub(crate) vars: Option<Vec<Description<'a>>>,
}

#[derive(Serialize, Debug)]
pub(crate) struct ToCompressModule<'a> {
    #[serde(rename = "Name")]
    pub(crate) name: &'a str,
    #[serde(rename = "Dir")]
    pub(crate) dir: &'a str,
    #[serde(rename = "Packages")]
    pub(crate) packages: Option<Vec<Description<'a>>>,
}

#[derive(Serialize, Debug)]
pub(crate) struct Description<'a> {
    pub name: &'a str,
    pub description: &'a str,
}

#[derive(Serialize, Debug)]
pub(crate) struct ToCompressVar<'a> {
    #[serde(rename = "Content")]
    pub(crate) content: &'a String,
    #[serde(rename = "Type", skip_serializing_if = "Option::is_none")]
    pub(crate) r#type: &'a Option<String>,
    #[serde(rename = "References")]
    pub(crate) refers: Vec<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct ToCompressFunc {
    #[serde(rename = "Content")]
    pub(crate) content: String,
    #[serde(rename = "Receiver", skip_serializing_if = "Option::is_none")]
    pub(crate) receiver: Option<String>,
    #[serde(rename = "Params", skip_serializing_if = "Option::is_none")]
    pub(crate) params: Option<Vec<KeyValueType>>,
    #[serde(rename = "Results", skip_serializing_if = "Option::is_none")]
    pub(crate) results: Option<Vec<KeyValueType>>,
    #[serde(rename = "Related_func")]
    pub(crate) related_func: Option<Vec<CalledType>>,
    #[serde(rename = "Related_type")]
    pub(crate) related_type: Option<Vec<KeyValueType>>,
    #[serde(rename = "Related_var")]
    pub(crate) related_var: Option<Vec<KeyValueType>>,
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

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct ToConvert {
    #[serde(rename = "Name")]
    pub(crate) name: String,
    #[serde(rename = "Receiver", skip_serializing_if = "Option::is_none")]
    pub(crate) receiver: Option<String>,
    #[serde(rename = "Definition")]
    pub(crate) definition: String,
    #[serde(rename = "Dependencies")]
    pub(crate) dependencies: HashMap<String, Reference>,
    #[serde(rename = "References")]
    pub(crate) references: HashMap<String, Reference>,
}

#[derive(Serialize, Deserialize, Debug)]
pub(crate) struct Reference {
    // #[serde(rename = "NeedMock")]
    // pub(crate) need_mock: bool,
    #[serde(rename = "Name", skip_serializing_if = "Option::is_none")]
    pub(crate) name: Option<String>,
    #[serde(rename = "Code")]
    pub(crate) code: String,
    #[serde(rename = "ImportPath", skip_serializing_if = "Option::is_none")]
    pub(crate) import: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ToMerge {
    #[serde(rename = "ID")]
    pub(crate) id: String,
    #[serde(rename = "Code")]
    pub(crate) code: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ToValidate {
    #[serde(rename = "Code")]
    pub(crate) code: String,
    #[serde(rename = "Error")]
    pub(crate) error: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Code {
    #[serde(rename = "Code")]
    pub code: String,
    #[serde(rename = "Imports", skip_serializing_if = "Option::is_none")]
    pub imports: Option<HashSet<String>>,
    #[serde(rename = "Crates", skip_serializing_if = "Option::is_none")]
    pub crates: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CodeCache {
    pub id: String,
    pub nodes: HashMap<String, Code>,
    #[serde(default)]
    pub files: HashMap<String, Code>,
}

impl CodeCache {
    pub fn new(id: String) -> Self {
        CodeCache {
            nodes: HashMap::new(),
            id: id,
            files: HashMap::new(),
        }
    }

    pub fn get(&self, id: &str) -> Option<&Code> {
        self.nodes.get(id)
    }

    pub fn get_by_id(&self, id: &Identity) -> Option<&Code> {
        self.nodes.get(&String::from(id))
    }

    pub fn insert(&mut self, id: &str, code: Code) {
        self.nodes.insert(id.to_string(), code);
    }

    pub fn insert_by_id(&mut self, id: &Identity, code: Code) {
        self.nodes.insert(String::from(id), code);
    }

    pub fn save_to_cache(&self) {
        let repo = serde_json::to_string(&self).expect("marshal struct error");
        get_cache()
            .put(&self.id, Vec::from(repo))
            .expect("save to cache failed");
    }

    pub fn load_from_cache(&mut self) -> bool {
        if let Some(repo) = get_cache().get(&self.id) {
            if let Ok(repo) = String::from_utf8(repo) {
                if let Ok(repo) = serde_json::from_str(&repo) {
                    *self = repo;
                    return true;
                }
            }
        }
        false
    }
}
