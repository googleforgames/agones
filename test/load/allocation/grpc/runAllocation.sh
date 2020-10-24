# Copyright 2020 Google LLC All Rights Reserved.
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

#!/bin/bash
# The number of times you want allocation test to be run
TESTRUNSCOUNT=3
NAMESPACE=default
EXTERNAL_IP=<IP_ADRESSS_TO_THE_ALLOCATOR_SERVICES_LOAD_BALANCER>
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

counter=1
while [ $counter -le $TESTRUNSCOUNT ]
do
    echo "Run number: " $counter
    go run main.go --ip ${EXTERNAL_IP} --port 443 --namespace ${NAMESPACE} --key ${KEY_FILE} --cert ${CERT_FILE} --cacert ${TLS_CA_FILE} --numberofclients $1 --perclientallocations $2 2>>./allocation_test_results.txt
    sleep 1200
    ((counter++))
done
