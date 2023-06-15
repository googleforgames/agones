fn main() {
    tonic_build::configure()
        // The SDK is just a client, no need to build the server types
        .build_server(false)
        .compile(
            &["proto/sdk/alpha/alpha.proto", "proto/sdk/sdk.proto"],
            &[
                "proto/googleapis",
                "proto/grpc-gateway",
                "proto/sdk/alpha",
                "proto/sdk",
            ],
        )
        .expect("failed to compile protobuffers");
}
