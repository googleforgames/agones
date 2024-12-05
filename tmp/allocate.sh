#!/bin/bash

# Get the key, cert, and tls files from "Send allocation request" instructions
# https://agones.dev/site/docs/advanced/allocator-service/
NAMESPACE=default
EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

echo "Starting go runs"

for _ in {1..10000}; do
  go run ../examples/allocator-client/main.go \
    --ip "${EXTERNAL_IP}" \
    --port 443 \
    --namespace "${NAMESPACE}" \
    --key "${KEY_FILE}" \
    --cert "${CERT_FILE}" \
    --cacert "${TLS_CA_FILE}"
done

echo "All done"
