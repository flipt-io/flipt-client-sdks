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

    let descriptor_path = get_cargo_target_dir()?.join("proto_descriptor.bin");

    prost_build::Config::new()
        .file_descriptor_set_path(&descriptor_path)
        .out_dir("src")
        .compile_well_known_types()
        .extern_path(".google.protobuf", "::pbjson_types")
        .compile_protos(&["proto/evaluation.proto"], &["proto"])
        .map_err(|e| format!("prost_build compilation failed: {e}"))?;

    let descriptor_set = std::fs::read(&descriptor_path)
        .unwrap_or_else(|e| panic!("Cannot read {:?}: {}", &descriptor_path, e));

    pbjson_build::Builder::new()
        .out_dir("gen/src")
        .register_descriptors(&descriptor_set)?
        .build(&[".flipt.evaluation"])
        .map_err(|e| format!("pbjson compilation failed: {e}"))?;

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
