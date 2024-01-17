extern crate cbindgen;

use std::env;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let crate_dir = env::var("CARGO_MANIFEST_DIR").unwrap();

    let generated_header_file = get_cargo_target_dir()?
        .join("release")
        .join("flipt_engine.h")
        .display()
        .to_string();

    cbindgen::Builder::new()
        .with_language(cbindgen::Language::C)
        .with_crate(crate_dir)
        .generate()
        .expect("Unable to generate bindings")
        .write_to_file(generated_header_file);

    Ok(())
}

// Ugly workaround to figure out target directory whilst running build scripts
// GH issue: https://github.com/rust-lang/cargo/issues/9661
fn get_cargo_target_dir() -> Result<std::path::PathBuf, Box<dyn std::error::Error>> {
    let skip_triple = std::env::var("TARGET")? == std::env::var("HOST")?;
    let skip_parent_dirs = if skip_triple { 4 } else { 5 };

    let out_dir = std::path::PathBuf::from(std::env::var("OUT_DIR")?);
    let mut current = out_dir.as_path();
    for _ in 0..skip_parent_dirs {
        current = current.parent().ok_or("not found")?;
    }

    Ok(std::path::PathBuf::from(current))
}
