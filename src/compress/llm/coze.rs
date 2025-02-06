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

use core::panic;
use reqwest::header::{HeaderMap, HeaderValue, ACCEPT, CONNECTION, CONTENT_TYPE, HOST};
use serde::{Deserialize, Serialize};
use serde_json::{from_str, to_string, to_string_pretty};
use std::ops::Add;
use std::{collections::HashMap, time::Duration};

use crate::{compress::llm::prompts::make_compress_prompt, config::CONFIG};

use super::ToCompress;

#[derive(Serialize, Deserialize, Debug)]
struct Message {
    role: String,
    r#type: String,
    content: String,
    content_type: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct StreamResponse {
    event: String,
    message: Message,
    conversation_id: String,
    index: i32,
    is_finish: bool,
}

#[derive(Serialize, Deserialize, Debug)]
struct Response {
    messages: Vec<Message>,
    conversation_id: String,
    code: i64,
    msg: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct BotQuery {
    bot_id: String,
    user: String,
    query: String,
    stream: bool,
}

fn panic_error(msg: &str, res: &reqwest::Response) {
    let headers: HashMap<String, String> = res
        .headers()
        .iter()
        .map(|(k, v)| (k.to_string(), String::from(v.to_str().unwrap_or_default())))
        .collect();
    panic!(
        "{}.\nstatus code is {}\nheader is {:?}\n",
        msg,
        res.status(),
        headers,
    );
}

fn make_lang_prompt() -> &'static str {
    match &CONFIG.language {
        crate::config::Language::Chinese => "现在，请使用中文解释以下输入:\n",
        crate::config::Language::English => {
            "Now, please use English to explain the following input:\n"
        }
    }
}

pub async fn coze_compress(to_compress: ToCompress) -> String {
    let mut headers = HeaderMap::new();

    let auth = format!("Bearer {}", CONFIG.coze_api_token.as_ref().unwrap());
    headers.insert(
        "Authorization",
        HeaderValue::from_str(auth.as_str()).unwrap(),
    );
    headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
    headers.insert(ACCEPT, HeaderValue::from_static("*/*"));
    headers.insert(CONNECTION, HeaderValue::from_static("keep-alive"));

    let client = reqwest::Client::new();

    let bot_id = CONFIG.coze_bot_id.as_ref().unwrap().clone();
    let to_compress_str = make_compress_prompt(&to_compress);
    let bot_query = BotQuery {
        bot_id: bot_id.to_string(),
        user: "welkey".to_string(),
        query: to_compress_str.to_string(),
        stream: true,
    };

    println!(
        "[coze_compress] request: {}",
        to_string(&bot_query).unwrap()
    );

    let rb = client
        .post("https://api.coze.com/open_api/v2/chat")
        .headers(headers)
        .json(&bot_query);

    let mut res = match rb.send().await {
        Ok(r) => r,
        Err(e) => {
            panic!("http request faield: {:?}", e);
        }
    };
    let status = res.status();
    if status != 200 {
        panic_error("coze request failed.", &res);
    }

    // streaming repsonse
    // allocate bytes buffer
    let mut sse_body = Vec::new();
    loop {
        match res.chunk().await {
            Ok(c) => {
                if let Some(chunk) = c {
                    // println!("chunk: {:?}", std::str::from_utf8(&chunk).unwrap());
                    sse_body.extend_from_slice(&chunk);
                } else {
                    break;
                }
            }
            Err(e) => {
                panic_error(format!("coze request failed: {:?}", e).as_str(), &res);
            }
        }
    }

    let sse_body = std::str::from_utf8(&sse_body).unwrap();
    let mut output: String = String::new();

    // handle SSE datas
    for line in sse_body.lines() {
        // println!("[coze_compress] receive chunk: {}", line);

        if line.len() == 0 || !line.starts_with("data:") {
            continue;
        }
        let data = line.strip_prefix("data:").unwrap().trim();
        if data.len() == 0 {
            continue;
        }
        let response: StreamResponse =
            from_str(data).expect(format!("{} is not a valid json", data).as_str());

        if response.is_finish || response.event != "message" {
            break;
        }

        if &response.message.r#type != "answer" {
            continue;
        }
        output += &response.message.content;
    }

    println!("[coze_compress] response body: {}", output);

    // unary response
    // let body = res.bytes().await.unwrap();
    // let resp: Response = from_str(std::str::from_utf8(&body).unwrap()).unwrap();
    // if resp.code != 0 {
    //     panic!("code is {}, msg is {}", resp.code, resp.msg);
    // }
    // for message in resp.messages {
    //     output += &message.content;
    // }

    output
}
