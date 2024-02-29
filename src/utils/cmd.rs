use std::io::{self, Error};
use std::process::Command;

pub fn run_command(cmd: &str, args: Vec<&str>) -> Result<String, Error> {
    let output = Command::new(cmd).args(args).output()?;

    match output.status.success() {
        true => Ok(String::from_utf8_lossy(&output.stdout).into_owned()),
        false => Err(io::Error::new(
            io::ErrorKind::Other,
            String::from_utf8_lossy(&output.stdout),
        )),
    }
}
