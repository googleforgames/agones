#!/bin/bash

# Generate the key, cert, and tls files from "Send allocation request" instructions
# https://agones.dev/site/docs/advanced/allocator-service/
NAMESPACE=default
EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

run_allocator_client () {
  go run ../examples/allocator-client/main.go \
    --ip "${EXTERNAL_IP}" \
    --port 443 \
    --namespace "${NAMESPACE}" \
    --key "${KEY_FILE}" \
    --cert "${CERT_FILE}" \
    --cacert "${TLS_CA_FILE}" \
    --requestsPerSecond 5 \
    --totalRequests 1000
}

echo "Starting go runs 1"
run_allocator_client &
sleep 10s

echo "Starting go runs 2"
run_allocator_client &
sleep 10s

echo "Starting go runs 3"
run_allocator_client &

wait
echo "All done"
