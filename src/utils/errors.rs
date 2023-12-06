// Import necessary types from the standard library
use std::{fmt, io, num::ParseIntError};
use std::process::Output;

// Our custom error type that can encompass various types of errors
// that may occur in our application
pub enum Error {
    Io(io::Error),
    GitCloneError { message: String },
    Parse(ParseIntError),
    // more error types can be added here as needed by the application...
}

// Implement the Display trait for our Error type, so we can print the error messages
impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match *self {
            Error::Io(ref err) => write!(f, "IO error: {}", err),
            Error::GitCloneError { ref message } => write!(f, "Git clone error: {}", message),
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
        Error::GitCloneError {
            message: String::from_utf8_lossy(&output.stderr).into_owned(),
        }
    }
}