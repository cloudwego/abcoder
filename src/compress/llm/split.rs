use text_splitter::{Characters, TextSplitter};

pub fn split_text(text: &str, max_length: usize) -> Vec<&str> {
    let splitter = TextSplitter::default()
        // Optionally can also have the splitter trim whitespace for you
        .with_trim_chunks(true);
    splitter.chunks(text, max_length).collect()
}
