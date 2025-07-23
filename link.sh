# PROJECT_ID="agones-sivasankaranr"
# BUCKET_NAME="test-ci"

PROJECT_ID="agones-images"
BUCKET_NAME="upgrade-test-container-logs"

echo "Logs from log bucket: https://console.cloud.google.com/logs/query;storageScope=storage,projects%2F${PROJECT_ID}%2Flocations%2Fglobal%2Fbuckets%2F${BUCKET_NAME}%2Fviews%2F_AllLogs;query=$(printf 'resource.labels.container_name=(\"upgrade-test-controller\" OR \"sdk-client-test\")' | jq -sRr @uri);cursorTimestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ);duration=PT10M?project=${PROJECT_ID}"


# PROJECT_ID="agones-images"
# BUCKET_NAME="upgrade-test-container-logs"

# # Query should NOT be URL-encoded at this stage
# query='resource.labels.container_name=("upgrade-test-controller" OR "sdk-client-test")'

# timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# log_url="https://console.cloud.google.com/logs/query;storageScope=storage,projects/${PROJECT_ID}/locations/global/buckets/${BUCKET_NAME}/views/_AllLogs?project=${PROJECT_ID}&query=$(printf '%s' "$query" | jq -sRr @uri)&cursorTimestamp=${timestamp}&duration=PT10M"

# echo "Logs from log bucket: $log_url"
