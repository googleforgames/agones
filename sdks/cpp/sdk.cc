// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "sdk.h"
#include "sdk.pb.h"

namespace agones {

    const int port = 59357;

    SDK::SDK() {
        channel = grpc::CreateChannel("localhost:" + std::to_string(port), grpc::InsecureChannelCredentials());
    }

    bool SDK::Connect() {
        if (!channel->WaitForConnected(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)))) {
            return false;
        }

        stub = stable::agones::dev::sdk::SDK::NewStub(channel);

        // make the health connection
        stable::agones::dev::sdk::Empty response;
        health = stub->Health(new grpc::ClientContext(), &response);

        return true;
    }

    grpc::Status SDK::Ready() {
        grpc::ClientContext *context = new grpc::ClientContext();
        context->set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)));
        stable::agones::dev::sdk::Empty request;
        stable::agones::dev::sdk::Empty response;

        return stub->Ready(context, request, &response);
    }

    bool SDK::Health() {
        stable::agones::dev::sdk::Empty request;
        return health->Write(request);
    }

    grpc::Status SDK::GameServer(stable::agones::dev::sdk::GameServer* response) {
        grpc::ClientContext *context = new grpc::ClientContext();
        context->set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)));
        stable::agones::dev::sdk::Empty request;

        return stub->GetGameServer(context, request, response);
    }

    grpc::Status SDK::WatchGameServer(const std::function<void(stable::agones::dev::sdk::GameServer)> callback) {
        grpc::ClientContext *context = new grpc::ClientContext();
        stable::agones::dev::sdk::Empty request;
        stable::agones::dev::sdk::GameServer gameServer;

        std::unique_ptr<grpc::ClientReader<stable::agones::dev::sdk::GameServer>> reader = stub->WatchGameServer(context, request);
        while (reader->Read(&gameServer)) {
            callback(gameServer);
        }
        return reader->Finish();
    }

    grpc::Status SDK::Shutdown() {
        grpc::ClientContext *context = new grpc::ClientContext();
        context->set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)));
        stable::agones::dev::sdk::Empty request;
        stable::agones::dev::sdk::Empty response;

        return stub->Shutdown(context, request, &response);
    }

    grpc::Status SDK::SetLabel(std::string key, std::string value) {
        grpc::ClientContext *context = new grpc::ClientContext();
        context->set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)));

        stable::agones::dev::sdk::KeyValue request;
        request.set_key(key);
        request.set_value(value);

        stable::agones::dev::sdk::Empty response;

        return stub->SetLabel(context, request, &response);
    }

    grpc::Status SDK::SetAnnotation(std::string key, std::string value) {
        grpc::ClientContext *context = new grpc::ClientContext();
        context->set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME), gpr_time_from_seconds(30, GPR_TIMESPAN)));

        stable::agones::dev::sdk::KeyValue request;
        request.set_key(key);
        request.set_value(value);

        stable::agones::dev::sdk::Empty response;

        return stub->SetAnnotation(context, request, &response);
    }
}