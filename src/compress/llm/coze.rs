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
    message: Message,
    conversation_id: String,
    index: i32,
    is_finish: bool,
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
            bot_id = "7332383249320214535";
            to_compress_str = t;
        }
        ToCompress::ToCompressFunc(f) => {
            bot_id = "7332383185508188178";
            to_compress_str = f;
        }
    }

    let bot_query = BotQuery {
        conversation_id: "123".to_string(),
        bot_id: bot_id.to_string(),
        user: "welkey".to_string(),
        query: to_compress_str.to_string(),
        stream: true,
    };

    let mut res = client
        .post(GLOBAL_COZE_API_URL.unwrap())
        .headers(headers)
        .json(&bot_query)
        .send()
        .await
        .unwrap();

    let status = res.status();
    if status != 200 {
        panic!(
            "status code is {}, body is {}",
            status,
            res.text().await.unwrap()
        );
    }

    let mut output: String = String::new();

    'outer: while let Some(chunk) = res.chunk().await.unwrap() {
        let sse_body = String::from_utf8(chunk.to_vec()).unwrap();
        // FIXME: the last chunk may be incompleted.
        let data = sse_body.split("event:message\ndata:").into_iter();

        for d in data {
            if d.trim().len() == 0 {
                continue;
            }
            let response: Response =
                from_str(d).expect(format!("{} is not a valid chunk", d).as_str());

            if response.is_finish {
                break 'outer;
            }

            if &response.message.r#type != "answer" {
                continue;
            }

            output += &response.message.content;
        }
    }

    output
}
