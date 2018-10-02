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

// A server that starts, and then stops after 60 seconds.
// This example really does nothing other than show how to integrate
// the C++ SDK.

#include <iostream>
#include <thread>
#include <agones/sdk.h>
#include <grpc++/grpc++.h>

// send health check pings
void doHealth(agones::SDK *sdk) {
    while (true) {
        if (!sdk->Health()) {
            std::cout << "Health ping failed" << std::endl;
        } else {
            std::cout << "Health ping sent" << std::endl;
        }
        std::this_thread::sleep_for(std::chrono::seconds(2));
    }
}

// watch GameServer Updates
void watchUpdates(agones::SDK *sdk) {
    std::cout << "Starting to watch GameServer updates..." << std::endl;
    sdk->WatchGameServer([](stable::agones::dev::sdk::GameServer gameserver){
        std::cout << "GameServer Update, name: " << gameserver.object_meta().name() << std::endl;
        std::cout << "GameServer Update, state: " << gameserver.status().state() << std::endl;
    });
}

int main() {
    std::cout << "C++ Game Server has started!" << std::endl;

    std::cout << "Getting the instance of the SDK!" << std::endl;
    agones::SDK *sdk = new agones::SDK();

    std::cout << "Attempting to connect..." << std::endl;
    bool connected = sdk->Connect();
    if (!connected) {
        std::cout << "Could not connect to the sidecar. Exiting!" << std::endl;
        return -1;
    }
    std::cout << "...handshake complete." << std::endl;

    std::thread health (doHealth, sdk);
    std::thread watch (watchUpdates, sdk);

    std::cout << "Setting a label" << std::endl;
    grpc::Status status = sdk->SetLabel("test-label", "test-value");
    if (!status.ok()) {
        std::cout << "Could not run SetLabel(): "+ status.error_message() + ". Exiting!" << std::endl;
        return -1;
    }

    std::cout << "Setting an annotation" << std::endl;
    status = sdk->SetAnnotation("test-annotation", "test value");
    if (!status.ok()) {
        std::cout << "Could not run SetAnnotation(): "+ status.error_message() + ". Exiting!" << std::endl;
        return -1;
    }

    std::cout << "Marking server as ready..." << std::endl;
    status = sdk->Ready();
    if (!status.ok()) {
        std::cout << "Could not run Ready(): "+ status.error_message() + ". Exiting!" << std::endl;
        return -1;
    }

    std::cout << "...marked Ready" << std::endl;

    std::cout << "Getting GameServer details..." << std::endl;
    stable::agones::dev::sdk::GameServer gameserver;
    status = sdk->GameServer(&gameserver);

    if (!status.ok()) {
        std::cout << "Could not run GameServer(): "+ status.error_message() + ". Exiting!" << std::endl;
        return -1;
    }

    std::cout << "GameServer name: " << gameserver.object_meta().name() << std::endl;

    for (int i = 0; i < 10; i++) {
        int time = i*10;
        std::cout << "Running for " + std::to_string(time) + " seconds !" << std::endl;

        std::this_thread::sleep_for(std::chrono::seconds(10));

        if (i == 5) {
            std::cout << "Shutting down after 60 seconds..." << std::endl;
            grpc::Status status = sdk->Shutdown();
            if (!status.ok()) {
                std::cout << "Could not run Shutdown():" + status.error_message() + ". Exiting!" << std::endl;
                return -1;
            }
            std::cout << "...marked for Shutdown" << std::endl;
        }
    }

    delete sdk;

    return 0;
}