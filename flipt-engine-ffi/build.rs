extern crate cbindgen;

use std::env;

use cbindgen::{Builder, Config, Language};

fn main() {
    let crate_dir = env::var("CARGO_MANIFEST_DIR").unwrap();
    Builder::new()
        .with_config(Config::from_root_or_default(std::path::Path::new(
            &crate_dir,
        )))
        .with_language(Language::C)
        .with_crate(crate_dir)
        .generate()
        .expect("Unable to generate bindings")
        .write_to_file("include/flipt_engine.h");
}
