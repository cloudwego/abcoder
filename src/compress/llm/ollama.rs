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

use serde::{Deserialize, Serialize};

use crate::{compress::llm::prompts::make_compress_prompt, config::CONFIG};

use super::ToCompress;

pub async fn ollama_compress(to_compress: ToCompress) -> String {
    let request_url = format!("http://localhost:11434/api/generate");
    let to_compress_str = make_compress_prompt(&to_compress);
    let model_name = CONFIG.ollama_model.as_ref().unwrap().clone();

    println!("use prompt:\n{}", to_compress_str);
    let req_body: OllamaReq = OllamaReq {
        model: model_name,
        prompt: to_compress_str,
    };
    let client = reqwest::Client::new();
    let mut response = client
        .post(&request_url)
        .json(&req_body)
        .send()
        .await
        .unwrap();

    let mut output = String::new();
    while let Ok(Some(chunk)) = response.chunk().await {
        let result = serde_json::from_slice(&chunk);
        if result.is_err() {
            break;
        }

        let value: OllamaResp = result.unwrap();

        if !value.response.is_empty() {
            output.push_str(value.response.as_str());
        }

        if value.done {
            break;
        }
    }

    output
}

#[derive(Serialize, Deserialize, Debug)]
struct OllamaReq {
    model: String,
    prompt: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct OllamaResp {
    model: String,
    created_at: String,
    response: String,
    done: bool,
}
