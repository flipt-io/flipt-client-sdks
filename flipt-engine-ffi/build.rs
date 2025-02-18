fn main() {
    println!("cargo:warning=Running build.rs");

    #[cfg(target_os = "linux")]
    {
        println!("cargo:warning=Detected Linux target, adding math and dl libs");
        println!("cargo:rustc-link-lib=dylib=m");
        println!("cargo:rustc-link-lib=dylib=dl");
    }
}
