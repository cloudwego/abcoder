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

use crate::config::CONFIG;
pub mod coze;
pub mod maas;
pub mod ollama;
pub mod prompts;

#[derive(Clone, Debug)]
pub enum ToCompress {
    ToCompressFunc(String),
    ToCompressType(String),
    ToCompressVar(String),
    ToCompressPkg(String),
    ToCompressModule(String),
    ToMergeRustPkg(String),
    ToValidateRust(String),
}

pub async fn compress(to_compress: &ToCompress) -> String {
    match CONFIG.api_type.as_str() {
        "coze" => coze::coze_compress(to_compress.clone()).await,
        "maas" => {
            if &CONFIG.mass_http_url == "" {
                maas::maas_compress_py(to_compress, CONFIG.maas_model_name.as_str())
            } else {
                maas::maas_compress_http(
                    to_compress,
                    CONFIG.maas_model_name.as_str(),
                    &CONFIG.mass_http_url,
                )
                .await
            }
        }
        "ollama" => ollama::ollama_compress(to_compress.clone()).await,
        _ => panic!("Unknown API type {}", CONFIG.api_type),
    }
}
