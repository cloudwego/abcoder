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

// Import necessary types from the standard library
use std::process::Output;
use std::{fmt, io, num::ParseIntError};

// Our custom error type that can encompass various types of errors
// that may occur in our application
pub enum Error {
    Io(io::Error),
    GitCloneError(String),
    Parse(String),
    // more error types can be added here as needed by the application...
}

// Implement the Display trait for our Error type, so we can print the error messages
impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match self {
            Error::Io(ref err) => write!(f, "IO error: {}", err),
            Error::GitCloneError(message) => write!(f, "Git clone error: {}", message),
            Error::Parse(ref err) => write!(f, "Parse error: {}", err),
            // ...
        }
    }
}

// We also implement the Debug trait, here we elect to delegate to the Display implementation
impl fmt::Debug for Error {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        fmt::Display::fmt(self, f)
    }
}

// We implement conversion from io:Error into our custom type
impl From<io::Error> for Error {
    fn from(err: io::Error) -> Error {
        Error::Io(err)
    }
}

// We implement conversion from Output into our custom type
impl From<Output> for Error {
    fn from(output: Output) -> Error {
        Error::GitCloneError(String::from_utf8_lossy(&output.stderr).into_owned())
    }
}
