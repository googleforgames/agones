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

#include <grpc++/grpc++.h>
#include "sdk.grpc.pb.h"

namespace agones {

    // The Agones SDK
    class SDK {
        public:
            // Creates a new instance of the SDK.
            // Does not connect to anything.
            SDK();

            // Must be called before any other functions on the SDK.
            // This will attempt to do a handshake with the sdk server, timing out
            // after 30 seconds.
            // Returns true if the connection was successful, false if not.
            bool Connect();

            // Marks the Game Server as ready to receive connections
            grpc::Status Ready();

            // Send Health ping. This is a synchronous request.
            bool Health();

            // Marks the Game Server as ready to shutdown
            grpc::Status Shutdown();

            ~SDK();

        private:
            std::shared_ptr<grpc::Channel> channel;
            std::unique_ptr<stable::agones::dev::sdk::SDK::Stub> stub;
            std::unique_ptr< ::grpc::ClientWriter< ::stable::agones::dev::sdk::Empty>> health;
    };
}
