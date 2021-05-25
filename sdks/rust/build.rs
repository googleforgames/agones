fn main() {
    println!("cargo:rerun-if-changed=build.rs");
    println!("cargo:rerun-if-changed=protos");

    tonic_build::configure()
        .compile(
            &["proto/sdk/alpha/alpha.proto", "proto/sdk/sdk.proto"],
            &["proto/googleapis", "proto/sdk/alpha", "proto/sdk"],
        )
        .expect("failed to compile protobuffers");
}
