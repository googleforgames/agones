// Copyright 2017 by the contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package healthcheck

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"time"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Example() {
	// Create a Handler that we can use to register liveness and readiness checks.
	health := NewHandler()

	// Add a readiness check to make sure an upstream dependency resolves in DNS.
	// If this fails we don't want to receive requests, but we shouldn't be
	// restarted or rescheduled.
	upstreamHost := "upstream.example.com"
	health.AddReadinessCheck(
		"upstream-dep-dns",
		DNSResolveCheck(upstreamHost, 50*time.Millisecond))

	// Add a liveness check to detect Goroutine leaks. If this fails we want
	// to be restarted/rescheduled.
	health.AddLivenessCheck("goroutine-threshold", GoroutineCountCheck(100))

	// Serve http://0.0.0.0:8080/live and http://0.0.0.0:8080/ready endpoints.
	// go http.ListenAndServe("0.0.0.0:8080", health)

	// Make a request to the readiness endpoint and print the response.
	fmt.Print(dumpRequest(health, "GET", "/ready"))

	// Output:
	// HTTP/1.1 503 Service Unavailable
	// Connection: close
	// Content-Type: application/json; charset=utf-8
	//
	// {}
}

func Example_database() {
	// Connect to a database/sql database
	var database *sql.DB
	database = connectToDatabase()

	// Create a Handler that we can use to register liveness and readiness checks.
	health := NewHandler()

	// Add a readiness check to we don't receive requests unless we can reach
	// the database with a ping in <1 second.
	health.AddReadinessCheck("database", DatabasePingCheck(database, 1*time.Second))

	// Serve http://0.0.0.0:8080/live and http://0.0.0.0:8080/ready endpoints.
	// go http.ListenAndServe("0.0.0.0:8080", health)

	// Make a request to the readiness endpoint and print the response.
	fmt.Print(dumpRequest(health, "GET", "/ready?full=1"))

	// Output:
	// HTTP/1.1 200 OK
	// Connection: close
	// Content-Type: application/json; charset=utf-8
	//
	// {
	//     "database": "OK"
	// }
}

func Example_advanced() {
	// Create a Handler that we can use to register liveness and readiness checks.
	health := NewHandler()

	// Make sure we can connect to an upstream dependency over TCP in less than
	// 50ms. Run this check asynchronously in the background every 10 seconds
	// instead of every time the /ready or /live endpoints are hit.
	//
	// Async is useful whenever a check is expensive (especially if it causes
	// load on upstream services).
	upstreamAddr := "upstream.example.com:5432"
	health.AddReadinessCheck(
		"upstream-dep-tcp",
		Async(TCPDialCheck(upstreamAddr, 50*time.Millisecond), 10*time.Second))

	// Add a readiness check against the health of an upstream HTTP dependency
	upstreamURL := "http://upstream-svc.example.com:8080/healthy"
	health.AddReadinessCheck(
		"upstream-dep-http",
		HTTPGetCheck(upstreamURL, 500*time.Millisecond))

	// Implement a custom check with a 50 millisecond timeout.
	health.AddLivenessCheck("custom-check-with-timeout", Timeout(func() error {
		// Simulate some work that could take a long time
		time.Sleep(time.Millisecond * 100)
		return nil
	}, 50*time.Millisecond))

	// Expose the readiness endpoints on a custom path /healthz mixed into
	// our main application mux.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	mux.HandleFunc("/healthz", health.ReadyEndpoint)

	// Sleep for just a moment to make sure our Async handler had a chance to run
	time.Sleep(500 * time.Millisecond)

	// Make a sample request to the /healthz endpoint and print the response.
	fmt.Println(dumpRequest(mux, "GET", "/healthz"))

	// Output:
	// HTTP/1.1 503 Service Unavailable
	// Connection: close
	// Content-Type: application/json; charset=utf-8
	//
	// {}
}

func Example_metrics() {
	// Create a new Prometheus registry (you'd likely already have one of these).
	registry := prometheus.NewRegistry()

	// Create a metrics-exposing Handler for the Prometheus registry
	// The healthcheck related metrics will be prefixed with the provided namespace
	health := NewMetricsHandler(registry, "example")

	// Add a simple readiness check that always fails.
	health.AddReadinessCheck("failing-check", func() error {
		return fmt.Errorf("example failure")
	})

	// Add a liveness check that always succeeds
	health.AddLivenessCheck("successful-check", func() error {
		return nil
	})

	// Create an "admin" listener on 0.0.0.0:9402
	adminMux := http.NewServeMux()
	// go http.ListenAndServe("0.0.0.0:9402", adminMux)

	// Expose prometheus metrics on /metrics
	adminMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Expose a liveness check on /live
	adminMux.HandleFunc("/live", health.LiveEndpoint)

	// Expose a readiness check on /ready
	adminMux.HandleFunc("/ready", health.ReadyEndpoint)

	// Make a request to the metrics endpoint and print the response.
	fmt.Println(dumpRequest(adminMux, "GET", "/metrics"))

	// Output:
	// HTTP/1.1 200 OK
	// Content-Length: 245
	// Content-Type: text/plain; version=0.0.4
	//
	// # HELP example_healthcheck_status Current check status (0 indicates success, 1 indicates failure)
	// # TYPE example_healthcheck_status gauge
	// example_healthcheck_status{check="failing-check"} 1
	// example_healthcheck_status{check="successful-check"} 0
}

func dumpRequest(handler http.Handler, method string, path string) string {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	dump, err := httputil.DumpResponse(rr.Result(), true)
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(dump), "\r\n", "\n", -1)
}

func connectToDatabase() *sql.DB {
	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	return db
}
