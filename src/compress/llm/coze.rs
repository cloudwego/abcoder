use std::ops::Add;

use reqwest::header::{HeaderMap, HeaderValue, ACCEPT, CONNECTION, CONTENT_TYPE, HOST};
use serde::{Deserialize, Serialize};
use serde_json::from_str;

use crate::compress::compress::ToCompress;

// TODO config-lize
static GLOBAL_COZE_TOKEN: Option<&str> = option_env!("coze_openapi_token");
static GLOBAL_COZE_API_URL: Option<&str> = option_env!("coze_openapi_url");

#[derive(Serialize, Deserialize, Debug)]
struct Message {
    role: String,
    r#type: String,
    content: String,
    content_type: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct Response {
    messages: Vec<Message>,
    conversation_id: String,
    code: i32,
    msg: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct BotQuery {
    conversation_id: String,
    bot_id: String,
    user: String,
    query: String,
    stream: bool,
}

pub async fn coze_compress(to_compress: ToCompress) -> String {
    let mut headers = HeaderMap::new();

    let auth = format!("Bearer {}", GLOBAL_COZE_TOKEN.unwrap());
    headers.insert(
        "Authorization",
        HeaderValue::from_str(auth.as_str()).unwrap(),
    );
    headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
    headers.insert(ACCEPT, HeaderValue::from_static("*/*"));
    headers.insert(CONNECTION, HeaderValue::from_static("keep-alive"));

    let client = reqwest::Client::new();

    let mut bot_id = "";
    let mut to_compress_str = String::new();

    match to_compress {
        ToCompress::ToCompressType(t) => {
            bot_id = "7325045807814852617";
            to_compress_str = t;
        }
        ToCompress::ToCompressFunc(f) => {
            bot_id = "7325049392581525554";
            to_compress_str = f;
        }
    }

    let bot_query = BotQuery {
        conversation_id: "123".to_string(),
        bot_id: bot_id.to_string(),
        user: "welkey".to_string(),
        query: to_compress_str.to_string(),
        stream: false,
    };

    let res = client
        .post(GLOBAL_COZE_API_URL.unwrap())
        .headers(headers)
        .json(&bot_query)
        .send()
        .await
        .unwrap();

    let res_text = res.text().await.unwrap();
    let response: Response = from_str(&res_text).unwrap();

    let mut output = String::new();

    for message in response.messages {
        if message.role == "assistant" && message.r#type == "answer" {
            output = message.content.to_string();
            break;
        }
    }

    output
}
