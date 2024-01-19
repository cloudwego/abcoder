use serde::{Deserialize, Serialize};

use crate::compress::compress::ToCompress;

pub async fn ollama_compress(to_compress: ToCompress) -> String {
    let request_url = format!("http://localhost:11434/api/generate");
    let mut model_name = "codellama-private";
    let mut to_compress_str = String::new();
    match to_compress {
        ToCompress::ToCompressType(t) => {
            model_name = "codellama-private-type";
            to_compress_str = t;
        }
        ToCompress::ToCompressFunc(f) => {
            to_compress_str = f;
        }
    }

    println!("use prompt:\n{}", to_compress_str);
    let req_body: OllamaReq = OllamaReq { model: model_name.to_string(), prompt: to_compress_str };
    let client = reqwest::Client::new();
    let mut response = client
        .post(&request_url)
        .json(&req_body)
        .send()
        .await.unwrap();


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