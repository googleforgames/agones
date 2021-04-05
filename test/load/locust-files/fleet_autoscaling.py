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

from locust import HttpLocust, TaskSet, events, task
import locust.events
import json
import os
import time
import socket
import atexit

FLEET_SIZE = 100
DEADLINE = 30 * 60


class UserBehavior(TaskSet):
    @task
    def scaleUpFleet(self):
        # Create a fleet.
        initial_size = 1
        start_time = time.time()
        payload = {
            "apiVersion": "agones.dev/v1",
            "kind": "Fleet",
            "metadata": {
                "generateName": "fleet-simple-game-server",
                "namespace": "default"
            },
            "spec": {
                "replicas": initial_size,
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
                                            "/simple-game-server:0.3")
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
        headers = {'content-type': 'application/json'}
        response = self.client.post(
            "/apis/agones.dev/v1/namespaces/default/fleets",
            data=json.dumps(payload),
            headers=headers)
        response_json = response.json()
        name = response_json['metadata']['name']
        selfLink = response_json['metadata']['selfLink']

        # Wait until the fleet is up.
        self.waitForScaling(selfLink, initial_size)
        total_time = int((time.time() - start_time) * 1000)
        events.request_success.fire(
            request_type="fleet_spawn_up",
            name="fleet_spawn_up",
            response_time=total_time,
            response_length=0)

        # Scale up the fleet.
        fleet_size = FLEET_SIZE
        resource_version = self.getResourceVersion(selfLink)
        start_time = time.time()
        payload = {
            "apiVersion": "agones.dev/v1",
            "kind": "Fleet",
            "metadata": {
                "name": str(name),
                "namespace": "default",
                "resourceVersion": str(resource_version)
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
                                "portPolicy": "dynamic",
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
                                            "/simple-game-server:0.3")
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
        response = self.client.put(
            selfLink,
            data=json.dumps(payload),
            headers=headers)
        self.waitForScaling(selfLink, fleet_size)
        total_time = int((time.time() - start_time) * 1000)
        events.request_success.fire(
            request_type="fleet_scaling_up",
            name="fleet_scaling_up",
            response_time=total_time,
            response_length=0)

        # Scale down the fleet.
        resource_version = self.getResourceVersion(selfLink)
        start_time = time.time()
        payload = {
            "apiVersion": "agones.dev/v1",
            "kind": "Fleet",
            "metadata": {
                "name": str(name),
                "namespace": "default",
                "resourceVersion": str(resource_version)
            },
            "spec": {
                "replicas": 0,
                "scheduling": "Packed",
                "strategy": {
                    "type": "RollingUpdate"
                },
                "template": {
                    "spec": {
                        "ports": [
                            {
                                "name": "default",
                                "portPolicy": "dynamic",
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
                                            "/simple-game-server:0.3")
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
        response = self.client.put(
            selfLink,
            data=json.dumps(payload),
            headers=headers)
        self.waitForScaling(selfLink, 0)
        total_time = int((time.time() - start_time) * 1000)
        events.request_success.fire(
            request_type="fleet_scaling_down",
            name="fleet_scaling_down",
            response_time=total_time,
            response_length=0)

        # Delete the fleet.
        response = self.client.delete(selfLink, headers=headers)

    def waitForScaling(self, selfLink, fleet_size):
        global ready_replicas
        start_time = time.time()
        while True:
            total_time = time.time() - start_time
            response = self.client.get(selfLink)
            response_json = response.json()
            status = response_json.get('status')
            if status is not None:
                ready_replicas = response_json['status']['readyReplicas']
            if (ready_replicas is not None and ready_replicas == fleet_size):
                print "Fleet is scaled to: " + str(fleet_size)
                break
            if (total_time > DEADLINE):
                print "Fleet did not scale up in time"
                events.request_success.fire(
                    request_type="fleet_scaling_timeout",
                    name="fleet_scaling_timeout",
                    response_time=total_time * 1000,
                    response_length=0)
                break

    def getResourceVersion(self, selfLink):
        response = self.client.get(selfLink)
        response_json = response.json()
        return response_json['metadata']['resourceVersion']


class AgonesUser(HttpLocust):
    task_set = UserBehavior
    min_wait = 500
    max_wait = 900

    def __init__(self):
        super(AgonesUser, self).__init__()
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
