use std::path::Path;
use std::process::Command;

use super::errors::Error;

// Function that clones a git repository and takes the URL of the
// repository and the directory where it should be cloned as arguments
pub fn git_clone(url: &str, directory: &Path) -> Result<(), Error> {
    // The git command is executed as a child process
    let output = Command::new("git")
        .args(&["clone", url, directory.to_str().unwrap()])
        .output()?;

    // Check if the git clone operation was successful
    if output.status.success() {
        Ok(())
    } else {
        // Convert the Output into our custom error type and return the error
        Err(output.into())
    }
}