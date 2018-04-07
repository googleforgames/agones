// Wrap in a new error
error_chain!{
    foreign_links {
        Grpc(::grpcio::Error);
    }

    errors {
        HealthPingConnectionFailure(t: String) {
            description("health ping connection failure"),
            display("health ping connection failure: '{}'", t),
        }
    }
}
