# Load and performance tests

Load tests aim to test the performance of the system under heavy load. For Agones, game server allocation is an example where heavy load and multiple parallel operations can be envisioned. Locust provides a good framework for testing a system under heavy load. It provides a light-weight mechanism to launch thousands of workers that run a given test.

The goal of performance tests is to provide metrics on various operations. For
Agones, fleet scaling is a good example where performance metrics are useful.
Similar to load tests, Locust can be used for performance tests with the main
difference being the number of workers that are launched.

## Build and run tests

Prerequisites:
- Docker.
- A running k8s cluster.

Load tests are written using Locust. These tests are also integrated with Graphite and Grafana
for storage and visualization of the results. 

### Running load tests using Locust on your local machine

This test uses the HTTP proxy on the local machine to access the k8s API. The default port for the proxy is 8001. To start a proxy to the Kubernetes API
server:

```
kubectl proxy [--port=PORT --address='0.0.0.0' --accept-hosts='.*'] &
```

Next, we need to build the Docker images and run the container:

```
docker build -t locust-files .
```

The above will build a Docker container to install Locust, Grafana, and Graphite and will configure
them. To run Locust tests for game server allocation:

```
docker run --rm --network="host" -e "LOCUST_FILE=gameserver_allocation.py"  -e "TARGET_HOST=http://127.0.0.1:8001" -p 8089:8089 locust-files:latest
```

To run Locust tests for fleet autoscaling:

```
docker run --rm --network="host" -e "LOCUST_FILE=fleet_autoscaling.py" -e "TARGET_HOST=http://127.0.0.1:8001" -p 8089:8089 locust-files:latest
```

NOTE: The Docker network host only works for Linux. For macOS and Windows you can use the special DNS name host.docker.internal. If you use Docker on macOS run this command instead:
```
docker run --rm -e "LOCUST_FILE=fleet_autoscaling.py" -e "TARGET_HOST=http:// host.docker.internal:8001" -p 8089:8089 locust-files:latest
```

After running the Docker container you can access Locust on port 8089 of your local machine. When running Locust tests, it is recommended to use the same value for number of users and hatch rate. For game server allocation, these numbers can be large, but for fleet autoscaling a single user is sufficient.

Grafana will be available on port 80, and Graphite on port 81.
