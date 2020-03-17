#!/bin/bash
# Provide the number of times you want allocation test to be run
testRunsCount=3
if [ -z "$1" ]
    then
        echo "No test run count provided, using default which is 3"
    else
        testRunsCount=$1
        if ! [[ $testRunsCount =~ ^[0-9]+$ ]] ; then
            echo "error: Not a positive number provided" >&2; exit 1
        fi
fi

counter=1
while [ $counter -le $testRunsCount ]
do
    echo "Run number: " $counter
    go run allocationload.go 2>>./allocation_test_results.txt
    sleep 500
    ((counter++))
done
