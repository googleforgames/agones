// This file is generated. Do not edit
// @generated

// https://github.com/Manishearth/rust-clippy/issues/702
#![allow(unknown_lints)]
#![allow(clippy)]

#![cfg_attr(rustfmt, rustfmt_skip)]

#![allow(box_pointers)]
#![allow(dead_code)]
#![allow(missing_docs)]
#![allow(non_camel_case_types)]
#![allow(non_snake_case)]
#![allow(non_upper_case_globals)]
#![allow(trivial_casts)]
#![allow(unsafe_code)]
#![allow(unused_imports)]
#![allow(unused_results)]

const METHOD_SDK_READY: ::grpcio::Method<super::sdk::Empty, super::sdk::Empty> = ::grpcio::Method {
    ty: ::grpcio::MethodType::Unary,
    name: "/stable.agones.dev.sdk.SDK/Ready",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

const METHOD_SDK_SHUTDOWN: ::grpcio::Method<super::sdk::Empty, super::sdk::Empty> = ::grpcio::Method {
    ty: ::grpcio::MethodType::Unary,
    name: "/stable.agones.dev.sdk.SDK/Shutdown",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

const METHOD_SDK_HEALTH: ::grpcio::Method<super::sdk::Empty, super::sdk::Empty> = ::grpcio::Method {
    ty: ::grpcio::MethodType::ClientStreaming,
    name: "/stable.agones.dev.sdk.SDK/Health",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

pub struct SdkClient {
    client: ::grpcio::Client,
}

impl SdkClient {
    pub fn new(channel: ::grpcio::Channel) -> Self {
        SdkClient {
            client: ::grpcio::Client::new(channel),
        }
    }

    pub fn ready_opt(&self, req: &super::sdk::Empty, opt: ::grpcio::CallOption) -> ::grpcio::Result<super::sdk::Empty> {
        self.client.unary_call(&METHOD_SDK_READY, req, opt)
    }

    pub fn ready(&self, req: &super::sdk::Empty) -> ::grpcio::Result<super::sdk::Empty> {
        self.ready_opt(req, ::grpcio::CallOption::default())
    }

    pub fn ready_async_opt(&self, req: &super::sdk::Empty, opt: ::grpcio::CallOption) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::sdk::Empty>> {
        self.client.unary_call_async(&METHOD_SDK_READY, req, opt)
    }

    pub fn ready_async(&self, req: &super::sdk::Empty) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::sdk::Empty>> {
        self.ready_async_opt(req, ::grpcio::CallOption::default())
    }

    pub fn shutdown_opt(&self, req: &super::sdk::Empty, opt: ::grpcio::CallOption) -> ::grpcio::Result<super::sdk::Empty> {
        self.client.unary_call(&METHOD_SDK_SHUTDOWN, req, opt)
    }

    pub fn shutdown(&self, req: &super::sdk::Empty) -> ::grpcio::Result<super::sdk::Empty> {
        self.shutdown_opt(req, ::grpcio::CallOption::default())
    }

    pub fn shutdown_async_opt(&self, req: &super::sdk::Empty, opt: ::grpcio::CallOption) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::sdk::Empty>> {
        self.client.unary_call_async(&METHOD_SDK_SHUTDOWN, req, opt)
    }

    pub fn shutdown_async(&self, req: &super::sdk::Empty) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::sdk::Empty>> {
        self.shutdown_async_opt(req, ::grpcio::CallOption::default())
    }

    pub fn health_opt(&self, opt: ::grpcio::CallOption) -> ::grpcio::Result<(::grpcio::ClientCStreamSender<super::sdk::Empty>, ::grpcio::ClientCStreamReceiver<super::sdk::Empty>)> {
        self.client.client_streaming(&METHOD_SDK_HEALTH, opt)
    }

    pub fn health(&self) -> ::grpcio::Result<(::grpcio::ClientCStreamSender<super::sdk::Empty>, ::grpcio::ClientCStreamReceiver<super::sdk::Empty>)> {
        self.health_opt(::grpcio::CallOption::default())
    }
    pub fn spawn<F>(&self, f: F) where F: ::futures::Future<Item = (), Error = ()> + Send + 'static {
        self.client.spawn(f)
    }
}

pub trait Sdk {
    fn ready(&self, ctx: ::grpcio::RpcContext, req: super::sdk::Empty, sink: ::grpcio::UnarySink<super::sdk::Empty>);
    fn shutdown(&self, ctx: ::grpcio::RpcContext, req: super::sdk::Empty, sink: ::grpcio::UnarySink<super::sdk::Empty>);
    fn health(&self, ctx: ::grpcio::RpcContext, stream: ::grpcio::RequestStream<super::sdk::Empty>, sink: ::grpcio::ClientStreamingSink<super::sdk::Empty>);
}

pub fn create_sdk<S: Sdk + Send + Clone + 'static>(s: S) -> ::grpcio::Service {
    let mut builder = ::grpcio::ServiceBuilder::new();
    let instance = s.clone();
    builder = builder.add_unary_handler(&METHOD_SDK_READY, move |ctx, req, resp| {
        instance.ready(ctx, req, resp)
    });
    let instance = s.clone();
    builder = builder.add_unary_handler(&METHOD_SDK_SHUTDOWN, move |ctx, req, resp| {
        instance.shutdown(ctx, req, resp)
    });
    let instance = s.clone();
    builder = builder.add_client_streaming_handler(&METHOD_SDK_HEALTH, move |ctx, req, resp| {
        instance.health(ctx, req, resp)
    });
    builder.build()
}
