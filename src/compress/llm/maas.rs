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

use std::{path::Path, process::Command, time::SystemTime};

use crate::config::CONFIG;

use super::{prompts::make_compress_prompt, ToCompress};

pub fn maas_compress_py(to_compress: &ToCompress, model_name: &str) -> String {
    let message = make_compress_prompt(to_compress);
    use std::io::Write;
    use std::process::{Command, Stdio};

    let command = "python3";
    let path = format!("{}/maas/model_call.py", &CONFIG.work_dir);
    println!("executing command: {} {} {}", command, path, model_name);

    // Create a new command process with piped stdin
    let mut child = Command::new(command)
        .arg(path)
        .arg(model_name)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .spawn()
        .expect("Failed to spawn command");

    // Write the message to the child process's stdin
    if let Some(mut stdin) = child.stdin.take() {
        stdin
            .write_all(message.as_bytes())
            .expect("Failed to write to stdin");
    }

    // Wait for the child process to complete
    let output = child.wait_with_output().expect("Failed to read stdout");
    // Check if the command was successful
    if output.status.success() {
        // Print the standard output
        let stdout = String::from_utf8_lossy(&output.stdout).to_string();
        println!("stdout: {}", stdout);
        return stdout;
    } else {
        // Print the standard error
        let stderr = String::from_utf8_lossy(&output.stderr);
        panic!("Error: {}", stderr);
    }
}

pub async fn maas_compress_http(to_compress: &ToCompress, model_name: &str, url: &str) -> String {
    let message = make_compress_prompt(to_compress);
    let url = format!("http://{}", url);
    let client = reqwest::Client::new();
    let response = client
        .post(&url)
        .json(&serde_json::json!({"model": model_name, "prompt": message}))
        .send()
        .await
        .expect("Failed to send request");

    if response.status().is_success() {
        let body = response.text().await.expect("Failed to read response body");
        // get json key "ans"
        let v: serde_json::Value = serde_json::from_str(&body).expect("Failed to parse response");
        if let Some(ans) = v.get("ans") {
            return ans.as_str().unwrap().to_string();
        } else {
            panic!("Failed to get key 'ans' from response");
        }
    } else {
        panic!("Failed to read response body");
    }
}
