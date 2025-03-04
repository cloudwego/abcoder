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

use std::io::{self, Error};
use std::process::Command;
use std::{env, os};

pub fn run_command(cmd: &str, args: Vec<&str>) -> Result<String, Error> {
    println!("execute command: {} {:?}", cmd, args);
    // get current directory
    let output = Command::new(cmd).args(args).output()?;

    match output.status.success() {
        true => Ok(String::from_utf8_lossy(&output.stdout).into_owned()),
        false => Err(io::Error::new(
            io::ErrorKind::Other,
            String::from_utf8_lossy(vec![output.stdout, output.stderr].concat().as_slice())
                .into_owned(),
        )),
    }
}

pub fn run_command_bytes(cmd: &str, args: Vec<String>) -> Result<Vec<u8>, Error> {
    println!("execute command: {} {:?}", cmd, args);
    let output = Command::new(cmd).args(args).output()?;

    match output.status.success() {
        true => Ok(output.stdout),
        false => Err(io::Error::new(
            io::ErrorKind::Other,
            String::from_utf8_lossy(vec![output.stdout, output.stderr].concat().as_slice())
                .into_owned(),
        )),
    }
}
