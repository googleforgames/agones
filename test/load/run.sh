#!/bin/bash

# TAKEN FROM: https://github.com/GoogleCloudPlatform/distributed-load-testing-using-kubernetes/blob/master/docker-image/locust-tasks/run.s

LOCUST="/usr/local/bin/locust"
LOCUS_OPTS="-f $LOCUST_FILE --host=$TARGET_HOST"
LOCUST_MODE=${LOCUST_MODE:-standalone}

if [[ "$LOCUST_MODE" = "master" ]]; then
    LOCUS_OPTS="$LOCUS_OPTS --master"
elif [[ "$LOCUST_MODE" = "worker" ]]; then
    LOCUS_OPTS="$LOCUS_OPTS --slave --master-host=$LOCUST_MASTER"
fi

echo "$LOCUST $LOCUS_OPTS"

$LOCUST $LOCUS_OPTS

