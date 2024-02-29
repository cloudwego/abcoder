use csv::Writer;
use serde_json::{json, Value};
use std::collections::HashMap;

use serde::{Deserialize, Serialize};

use crate::storage::cache::get_cache;

#[derive(Serialize, Deserialize, Debug)]
pub struct Repository {
    pub id: Option<String>,
    #[serde(rename = "ModName")]
    pub(crate) mod_name: String,
    #[serde(rename = "Packages")]
    pub packages: HashMap<String, Package>,
}

impl Repository {
    pub fn merge_with(&mut self, other: &Repository) {
        for (pkg_name, pkg) in other.packages.iter() {
            if let Some(p) = self.packages.get_mut(pkg_name) {
                p.merge_with(pkg);
            } else {
                self.packages.insert(pkg_name.clone(), pkg.clone());
            }
        }
    }

    pub fn save_to_cache(&self) {
        let repo = serde_json::to_string(&self).expect("marshal struct error");
        get_cache()
            .put(self.id.as_ref().unwrap(), Vec::from(repo))
            .expect("save to cache failed");
    }

    pub fn get_func(&self, id: &Identity) -> Option<&Function> {
        if let Some(pkg) = self.packages.get(&id.pkg_path) {
            if let Some(f) = pkg.functions.get(&id.name) {
                return Some(f);
            }
        }
        None
    }

    pub fn get_type(&self, id: &Identity) -> Option<&Struct> {
        if let Some(pkg) = self.packages.get(&id.pkg_path) {
            if let Some(f) = pkg.types.get(&id.name) {
                return Some(f);
            }
        }
        None
    }

    pub fn to_csv_summary(&self) -> String {
        let mut w = Writer::from_writer(Vec::new());
        // add header
        w.write_record(&["Package", "Name", "Kind", "Signature", "Summary"])
            .unwrap();
        for (pname, pkg) in self.packages.iter() {
            for (name, f) in pkg.functions.iter() {
                let tmp = &"".to_string();
                let summary = f.compress_data.as_ref().unwrap_or(tmp);
                // split content, 1024B for each
                let mut start = 0;
                let mut end = 1024;
                while start < summary.len() {
                    if end > summary.len() {
                        end = summary.len();
                    }
                    if start >= 1024 {
                        start -= 100;
                    }
                    w.write_record(&[
                        pname,
                        name,
                        "Function",
                        f.content.split_once('\n').unwrap_or((&f.content, "")).0,
                        &summary[start..end],
                    ])
                    .unwrap();
                    start = end;
                    end += 924;
                }
            }
            for (name, t) in pkg.types.iter() {
                let tmp = &"".to_string();
                let summary = t.compress_data.as_ref().unwrap_or(tmp);
                // split content, 1024B for each
                let mut start = 0;
                let mut end = 1024;
                while start < summary.len() {
                    if end > summary.len() {
                        end = summary.len();
                    }
                    if start >= 1024 {
                        start -= 100;
                    }
                    w.write_record(&[
                        pname,
                        name,
                        "Type",
                        t.content.split_once('\n').unwrap_or((&t.content, "")).0,
                        &summary[start..end],
                    ])
                    .unwrap();
                    start = end;
                    end += 924;
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
        for (pname, pkg) in self.packages.iter() {
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
        }
        w.flush().unwrap();
        String::from_utf8(w.into_inner().unwrap()).unwrap()
    }

    pub fn to_csv_pkgs(&self) -> String {
        let mut w = Writer::from_writer(Vec::new());
        // add header
        w.write_record(&["Name", "Summary", "Content"]).unwrap();
        for (pname, pkg) in self.packages.iter() {
            w.write_record(&[
                &format!("{}", pname),
                pkg.compress_data.as_ref().unwrap_or(&"".to_string()),
                &pkg.export_api(),
            ])
            .unwrap();
        }
        w.flush().unwrap();
        String::from_utf8(w.into_inner().unwrap()).unwrap()
    }
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
        if self.id != other.id {
            self.id = other.id.clone();
        }
        // if self.dependencies.is_empty() && !other.dependencies.is_empty() {
        //     self.dependencies = other.dependencies.clone();
        // }
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
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "IsMethod")]
    is_method: bool,
    #[serde(rename = "PkgPath")]
    pub pkg_path: String,
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Content")]
    pub content: String,
    #[serde(rename = "AssociatedStruct")]
    associated_struct: Option<Identity>,
    #[serde(rename = "InternalFunctionCalls")]
    pub internal_function_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "ThirdPartyFunctionCalls")]
    pub third_party_function_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "InternalMethodCalls")]
    pub internal_method_calls: Option<HashMap<String, Identity>>,
    #[serde(rename = "ThirdPartyMethodCalls")]
    pub third_party_method_calls: Option<HashMap<String, Identity>>,

    // compress_data
    pub compress_data: Option<String>,
}

impl Function {
    pub fn merge_with(&mut self, other: &Function) {
        if self.is_exported != other.is_exported {
            self.is_exported = other.is_exported;
        }
        if self.is_method == false && other.is_method == true {
            self.is_method = true;
        }
        if self.pkg_path.is_empty() && !other.pkg_path.is_empty() {
            self.pkg_path = other.pkg_path.clone();
        }
        if self.name.is_empty() && !other.name.is_empty() {
            self.name = other.name.clone();
        }
        if self.content.is_empty() && !other.content.is_empty() {
            self.content = other.content.clone();
        }
        if self.associated_struct.is_none() && other.associated_struct.is_some() {
            self.associated_struct = other.associated_struct.clone();
        }
        if self.internal_function_calls.is_none() && other.internal_function_calls.is_some() {
            self.internal_function_calls = other.internal_function_calls.clone();
        }
        if self.third_party_function_calls.is_none() && other.third_party_function_calls.is_some() {
            self.third_party_function_calls = other.third_party_function_calls.clone();
        }
        if self.internal_method_calls.is_none() && other.internal_method_calls.is_some() {
            self.internal_method_calls = other.internal_method_calls.clone();
        }
        if self.third_party_method_calls.is_none() && other.third_party_method_calls.is_some() {
            self.third_party_method_calls = other.third_party_method_calls.clone();
        }
        if self.compress_data.is_none() && other.compress_data.is_some() {
            self.compress_data = other.compress_data.clone();
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Struct {
    #[serde(rename = "Exported", default)]
    pub is_exported: bool,
    #[serde(rename = "TypeKind")]
    type_kind: u8,
    #[serde(rename = "PkgPath")]
    pub(crate) pkg_path: String,
    #[serde(rename = "Name")]
    pub(crate) name: String,
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
    pub fn merge_with(&mut self, other: &Struct) {
        if self.is_exported != other.is_exported {
            self.is_exported = other.is_exported;
        }
        self.type_kind = other.type_kind;
        if self.pkg_path.is_empty() && !other.pkg_path.is_empty() {
            self.pkg_path = other.pkg_path.clone();
        }
        if self.name.is_empty() && !other.name.is_empty() {
            self.name = other.name.clone();
        }
        if self.content.is_empty() && !other.content.is_empty() {
            self.content = other.content.clone();
        }
        if self.sub_struct.is_none() && other.sub_struct.is_some() {
            self.sub_struct = other.sub_struct.clone();
        }
        if self.inline_struct.is_none() && other.inline_struct.is_some() {
            self.inline_struct = other.inline_struct.clone();
        }
        if self.methods.is_none() && other.methods.is_some() {
            self.methods = other.methods.clone();
        }
        if self.compress_data.is_none() && other.compress_data.is_some() {
            self.compress_data = other.compress_data.clone();
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Identity {
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
