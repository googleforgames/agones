# healthcheck
[![Build Status](https://travis-ci.org/heptiolabs/healthcheck.svg?branch=master)](https://travis-ci.org/heptiolabs/healthcheck)
[![Go Report Card](https://goreportcard.com/badge/github.com/heptiolabs/healthcheck)](https://goreportcard.com/report/github.com/heptiolabs/healthcheck)
[![GoDoc](https://godoc.org/github.com/heptiolabs/healthcheck?status.svg)](https://godoc.org/github.com/heptiolabs/healthcheck)

Healthcheck is a library for implementing Kubernetes [liveness and readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) probe handlers in your Go application.

## Features

 - Integrates easily with Kubernetes. This library explicitly separates liveness vs. readiness checks instead of lumping everything into a single category of check.

 - Optionally exposes each check as a [Prometheus gauge](https://prometheus.io/docs/concepts/metric_types/#gauge) metric. This allows for cluster-wide monitoring and alerting on individual checks.

 - Supports asynchronous checks, which run in a background goroutine at a fixed interval. These are useful for expensive checks that you don't want to add latency to the liveness and readiness endpoints.

 - Includes a small library of generically useful checks for validating upstream DNS, TCP, HTTP, and database dependencies as well as checking basic health of the Go runtime.

## Usage

See the [GoDoc examples](https://godoc.org/github.com/heptiolabs/healthcheck) for more detail.

 - Install with `go get` or your favorite Go dependency manager: `go get -u github.com/heptiolabs/healthcheck`

 - Import the package: `import "github.com/heptiolabs/healthcheck"`

 - Create a `healthcheck.Handler`:
   ```go
   health := healthcheck.NewHandler()
   ```

 - Configure some application-specific liveness checks (whether the app itself is unhealthy):
   ```go
   // Our app is not happy if we've got more than 100 goroutines running.
   health.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(100))
   ```

 - Configure some application-specific readiness checks (whether the app is ready to serve requests):
   ```go
   // Our app is not ready if we can't resolve our upstream dependency in DNS.
   health.AddReadinessCheck(
       "upstream-dep-dns",
       healthcheck.DNSResolveCheck("upstream.example.com", 50*time.Millisecond))

   // Our app is not ready if we can't connect to our database (`var db *sql.DB`) in <1s.
   health.AddReadinessCheck("database", healthcheck.DatabasePingCheck(db, 1*time.Second))
   ```

 - Expose the `/live` and `/ready` endpoints over HTTP (on port 8086):
   ```go
   go http.ListenAndServe("0.0.0.0:8086", health)
   ```

 - Configure your Kubernetes container with HTTP liveness and readiness probes see the ([Kubernetes documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/)) for more detail:
   ```yaml
   # this is a bare bones example
   # copy and paste livenessProbe and readinessProbe as appropriate for your app
   apiVersion: v1
   kind: Pod
   metadata:
     name: heptio-healthcheck-example
   spec:
     containers:
     - name: liveness
       image: your-registry/your-container

       # define a liveness probe that checks every 5 seconds, starting after 5 seconds
       livenessProbe:
         httpGet:
           path: /live
           port: 8086
         initialDelaySeconds: 5
         periodSeconds: 5

       # define a readiness probe that checks every 5 seconds
       readinessProbe:
         httpGet:
           path: /ready
           port: 8086
         periodSeconds: 5
   ```

 - If one of your readiness checks fails, Kubernetes will stop routing traffic to that pod within a few seconds (depending on `periodSeconds` and other factors).

 - If one of your liveness checks fails or your app becomes totally unresponsive, Kubernetes will restart your container.

 ## HTTP Endpoints
 When you run `go http.ListenAndServe("0.0.0.0:8086", health)`, two HTTP endpoints are exposed:

  - **`/live`**: liveness endpoint (HTTP 200 if healthy, HTTP 503 if unhealthy)
  - **`/ready`**: readiness endpoint (HTTP 200 if healthy, HTTP 503 if unhealthy)

Pass the `?full=1` query parameter to see the full check results as JSON. These are omitted by default for performance.