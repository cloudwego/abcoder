use std::fs;
use std::path::Path;

pub fn get_all_md(dir: &Path, files_list: &mut Vec<String>) -> std::io::Result<()> {
    if dir.is_dir() {
        for entry in fs::read_dir(dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_dir() && !path.ends_with(".github") {
                get_all_md(&path, files_list)?;
            } else if path.is_file() {
                if let Some(ext) = path.extension() {
                    if ext == "md" {
                        files_list.push(path.to_string_lossy().to_string());
                    }
                }
            }
        }
    }
    Ok(())
}
