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
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Package {
    #[serde(rename = "Functions")]
    pub functions: HashMap<String, Function>,
    #[serde(rename = "Types")]
    pub types: HashMap<String, Struct>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Function {
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

#[derive(Serialize, Deserialize, Debug)]
pub struct Struct {
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

#[derive(Serialize, Deserialize, Debug)]
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
