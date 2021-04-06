# Copyright 2018 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from locust import HttpLocust, TaskSet, clients, events, task
import locust.events
import json
import time
import socket
import atexit

FLEET_SIZE = 100
FLEET_NAME = "scale-test-fleet"
FLEET_RESOURCE_PATH = (
    "/apis/agones.dev/v1/namespaces/default/fleets")
ALLOCATION_RESOURCE_PATH = (
    "/apis/allocation.agones.dev/v1/namespaces/default"
    "/gameserverallocations")


class UserBehavior(TaskSet):
    @task
    def allocate_game_server(self):
        # Get the number of ready game servers.
        fleet_self_link = FLEET_RESOURCE_PATH + "/" + FLEET_NAME
        response = self.client.get(str(fleet_self_link))
        response_json = response.json()
        ready_replicas = response_json['status']['readyReplicas']
        # Allocate game servers.
        payload = {
            "apiVersion": "allocation.agones.dev/v1",
            "kind": "GameServerAllocation",
            "metadata": {
                "generateName": "gs-allocation-",
                "namespace": "default"
            },
            "spec": {
                "required": {
                    "matchLabels": {
                        "agones.dev/fleet": FLEET_NAME
                    }
                }
            }
        }
        headers = {'content-type': 'application/json'}
        start_time = time.time()
        response = self.client.post(
            str(ALLOCATION_RESOURCE_PATH),
            data=json.dumps(payload),
            headers=headers)
        events.request_success.fire(
            request_type="ReadyReplicas",
            name="ReadyReplicas",
            response_time=ready_replicas,
            response_length=0)
        # Wait for allocation.
        while True:
            response_json = response.json()
            status = response_json.get('status')
            if (status is not None):
                game_server_state = response_json['status']['state']
                if (game_server_state == "Allocated"):
                    total_time = int((time.time() - start_time) * 1000)
                    events.request_success.fire(
                        request_type="GameServerAllocated",
                        name="GameServerAllocated",
                        response_time=total_time,
                        response_length=0)
                    break
                elif (game_server_state == "UnAllocated"):
                    total_time = int((time.time() - start_time) * 1000)
                    events.request_success.fire(
                        request_type="GameServerUnAllocated",
                        name="GameServerUnAllocated",
                        response_time=total_time,
                        response_length=0)
                    break
            else:
                self_link = response_json['metadata']['selfLink']
                response = self.client.get(self_link)


class AgonesUser(HttpLocust):
    def setup(self):
        # Create a fleet.
        client = clients.HttpSession(base_url=self.host)
        self.create_fleet(client, FLEET_NAME, FLEET_SIZE)

    def create_fleet(self, client, fleet_name, fleet_size):
        # Create a Fleet and wait for it to scale up.
        print "Creating Fleet: " + fleet_name
        payload = {
            "apiVersion": "agones.dev/v1",
            "kind": "Fleet",
            "metadata": {
                "name": str(fleet_name),
                "namespace": "default"
            },
            "spec": {
                "replicas": fleet_size,
                "scheduling": "Packed",
                "strategy": {
                    "type": "RollingUpdate"
                },
                "template": {
                    "spec": {
                        "ports": [
                            {
                                "name": "default",
                                "portPolicy": "Dynamic",
                                "containerPort": 26000
                            }
                        ],
                        "template": {
                            "spec": {
                                "containers": [
                                    {
                                        "name": "simple-game-server",
                                        "image": (
                                            "gcr.io/agones-images"
                                            "/simple-game-server:0.3"),
                                        "resources": {
                                            "limits": {
                                                "cpu": "20m",
                                                "memory": "64Mi"
                                            },
                                            "requests": {
                                                "cpu": "20m",
                                                "memory": "64Mi"
                                            }
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
        headers = {'content-type': 'application/json'}
        response = client.post(
            str(FLEET_RESOURCE_PATH),
            data=json.dumps(payload),
            headers=headers)
        response_json = response.json()
        self_link = response_json['metadata']['selfLink']
        self.wait_for_scaling(client, self_link, fleet_size)

    def wait_for_scaling(self, client, self_link, fleet_size):
        global ready_replicas
        while True:
            response = client.get(self_link)
            response_json = response.json()
            status = response_json.get('status')
            if status is not None:
                ready_replicas = response_json['status']['readyReplicas']
            if (ready_replicas is not None and ready_replicas == fleet_size):
                print "Fleet is scaled to: " + str(fleet_size)
                break

    task_set = UserBehavior
    min_wait = 500
    max_wait = 900

    def __init__(self):
        super(AgonesUser, self).__init__()
        # Create socket to send metrics to Grafana.
        self.sock = socket.socket()
        self.sock.connect(("localhost", 2003))
        locust.events.request_success += self.hook_request_success
        atexit.register(self.exit_handler)

    def hook_request_success(self,
                             request_type,
                             name,
                             response_time,
                             response_length):
        self.sock.send(
            "%s %d %d\n" % (
                "performance." + name.replace('.', '-'),
                response_time,
                time.time()))

    def exit_handler(self):
        self.sock.shutdown(socket.SHUT_RDWR)
        self.sock.close()

    def delete_fleet(self, fleet_name):
        # Delete the Fleet.
        headers = {'content-type': 'application/json'}
        self_link = FLEET_RESOURCE_PATH + str(fleet_name)
        self.client.delete(self_link, headers=headers)
