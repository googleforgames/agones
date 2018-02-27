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
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"
)

// TCPDialCheck returns a Check that checks TCP connectivity to the provided
// endpoint.
func TCPDialCheck(addr string, timeout time.Duration) Check {
	return func() error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}

// HTTPGetCheck returns a Check that performs an HTTP GET request against the
// specified URL. The check fails if the response times out or returns a non-200
// status code.
func HTTPGetCheck(url string, timeout time.Duration) Check {
	client := http.Client{
		Timeout: timeout,
		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return func() error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("returned status %d", resp.StatusCode)
		}
		return nil
	}
}

// DatabasePingCheck returns a Check that validates connectivity to a
// database/sql.DB using Ping().
func DatabasePingCheck(database *sql.DB, timeout time.Duration) Check {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if database == nil {
			return fmt.Errorf("database is nil")
		}
		return database.PingContext(ctx)
	}
}

// DNSResolveCheck returns a Check that makes sure the provided host can resolve
// to at least one IP address within the specified timeout.
func DNSResolveCheck(host string, timeout time.Duration) Check {
	resolver := net.Resolver{}
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		addrs, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return err
		}
		if len(addrs) < 1 {
			return fmt.Errorf("could not resolve host")
		}
		return nil
	}
}

// GoroutineCountCheck returns a Check that fails if too many goroutines are
// running (which could indicate a resource leak).
func GoroutineCountCheck(threshold int) Check {
	return func() error {
		count := runtime.NumGoroutine()
		if count > threshold {
			return fmt.Errorf("too many goroutines (%d > %d)", count, threshold)
		}
		return nil
	}
}
