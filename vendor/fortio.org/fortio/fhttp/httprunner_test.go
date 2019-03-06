// Copyright 2017 Istio Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package fhttp

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"fortio.org/fortio/log"
)

func TestHTTPRunner(t *testing.T) {
	mux, addr := DynamicHTTPServer(false)
	mux.HandleFunc("/foo/", EchoHandler)
	baseURL := fmt.Sprintf("http://localhost:%d/", addr.Port)

	opts := HTTPRunnerOptions{}
	opts.QPS = 100
	opts.URL = baseURL
	opts.DisableFastClient = true
	_, err := RunHTTPTest(&opts)
	if err == nil {
		t.Error("Expecting an error but didn't get it when not using full url")
	}
	opts.DisableFastClient = false
	opts.URL = baseURL + "foo/bar?delay=20ms&status=200:100"
	opts.Profiler = "test.profile"
	res, err := RunHTTPTest(&opts)
	if err != nil {
		t.Error(err)
		return
	}
	totalReq := res.DurationHistogram.Count
	httpOk := res.RetCodes[http.StatusOK]
	if totalReq != httpOk {
		t.Errorf("Mismatch between requests %d and ok %v", totalReq, res.RetCodes)
	}
	if res.SocketCount != res.RunnerResults.NumThreads {
		t.Errorf("%d socket used, expected same as thread# %d", res.SocketCount, res.RunnerResults.NumThreads)
	}
	// Test raw client, should get warning about non init timeout:
	rawOpts := HTTPOptions{
		URL: opts.URL,
	}
	o1 := rawOpts
	if r, _, _ := NewFastClient(&o1).Fetch(); r != http.StatusOK {
		t.Errorf("Fast Client with raw option should still work with warning in logs")
	}
	o1 = rawOpts
	o1.URL = "http://www.doesnotexist.badtld/"
	c := NewStdClient(&o1)
	c.ChangeURL(rawOpts.URL)
	if r, _, _ := c.Fetch(); r != http.StatusOK {
		t.Errorf("Std Client with raw option should still work with warning in logs")
	}
}

func testHTTPNotLeaking(t *testing.T, opts *HTTPRunnerOptions) {
	ngBefore1 := runtime.NumGoroutine()
	t.Logf("Number go routine before test %d", ngBefore1)
	mux, addr := DynamicHTTPServer(false)
	mux.HandleFunc("/echo100", EchoHandler)
	url := fmt.Sprintf("http://localhost:%d/echo100", addr.Port)
	numCalls := 100
	opts.NumThreads = numCalls / 2 // make 2 calls per thread
	opts.Exactly = int64(numCalls)
	opts.QPS = float64(numCalls) / 2 // take 1 second
	opts.URL = url
	// Warm up round 1
	res, err := RunHTTPTest(opts)
	if err != nil {
		t.Error(err)
		return
	}
	httpOk := res.RetCodes[http.StatusOK]
	if opts.Exactly != httpOk {
		t.Errorf("Run1: Mismatch between requested calls %d and ok %v", numCalls, res.RetCodes)
	}
	ngBefore2 := runtime.NumGoroutine()
	t.Logf("Number of go routine after warm up / before 2nd test %d", ngBefore2)
	// 2nd run, should be stable number of go routines after first, not keep growing:
	res, err = RunHTTPTest(opts)
	// it takes a while for the connections to close with std client (!) why isn't CloseIdleConnections() synchronous
	runtime.GC()
	runtime.GC() // 2x to clean up more... (#178)
	ngAfter := runtime.NumGoroutine()
	t.Logf("Number of go routine after 2nd test %d", ngAfter)
	if err != nil {
		t.Error(err)
		return
	}
	httpOk = res.RetCodes[http.StatusOK]
	if opts.Exactly != httpOk {
		t.Errorf("Run2: Mismatch between requested calls %d and ok %v", numCalls, res.RetCodes)
	}
	// allow for ~8 goroutine variance, as we use 50 if we leak it will show (was failing before #167)
	if ngAfter > ngBefore2+8 {
		t.Errorf("Goroutines after test %d, expected it to stay near %d", ngAfter, ngBefore2)
	}
	if !opts.DisableFastClient {
		// only fast client so far has a socket count
		if res.SocketCount != res.RunnerResults.NumThreads {
			t.Errorf("%d socket used, expected same as thread# %d", res.SocketCount, res.RunnerResults.NumThreads)
		}
	}
}

func TestHttpNotLeakingFastClient(t *testing.T) {
	testHTTPNotLeaking(t, &HTTPRunnerOptions{})
}

func TestHttpNotLeakingStdClient(t *testing.T) {
	testHTTPNotLeaking(t, &HTTPRunnerOptions{HTTPOptions: HTTPOptions{DisableFastClient: true}})
}

func TestHTTPRunnerClientRace(t *testing.T) {
	mux, addr := DynamicHTTPServer(false)
	mux.HandleFunc("/echo1/", EchoHandler)
	URL := fmt.Sprintf("http://localhost:%d/echo1/", addr.Port)

	opts := HTTPRunnerOptions{}
	opts.Init(URL)
	opts.QPS = 100
	opts2 := opts
	go RunHTTPTest(&opts2)
	res, err := RunHTTPTest(&opts)
	if err != nil {
		t.Error(err)
		return
	}
	totalReq := res.DurationHistogram.Count
	httpOk := res.RetCodes[http.StatusOK]
	if totalReq != httpOk {
		t.Errorf("Mismatch between requests %d and ok %v", totalReq, res.RetCodes)
	}
}

func TestClosingAndSocketCount(t *testing.T) {
	mux, addr := DynamicHTTPServer(false)
	mux.HandleFunc("/echo42/", EchoHandler)
	URL := fmt.Sprintf("http://localhost:%d/echo42/?close=1", addr.Port)
	opts := HTTPRunnerOptions{}
	opts.Init(URL)
	opts.QPS = 10
	numReq := int64(50) // can't do too many without running out of fds on mac
	opts.Exactly = numReq
	opts.NumThreads = 5
	res, err := RunHTTPTest(&opts)
	if err != nil {
		t.Fatal(err)
	}
	totalReq := res.DurationHistogram.Count
	if totalReq != numReq {
		t.Errorf("Mismatch between requests %d and expected %d", totalReq, numReq)
	}
	httpOk := res.RetCodes[http.StatusOK]
	if totalReq != httpOk {
		t.Errorf("Mismatch between requests %d and ok %v", totalReq, res.RetCodes)
	}
	if int64(res.SocketCount) != numReq {
		t.Errorf("When closing, got %d while expected as many sockets as requests %d", res.SocketCount, numReq)
	}
}

func TestHTTPRunnerBadServer(t *testing.T) {
	// Using http to an https server (or the current 'close all' dummy https server)
	// should fail:
	_, addr := DynamicHTTPServer(true)
	baseURL := fmt.Sprintf("http://localhost:%d/", addr.Port)

	opts := HTTPRunnerOptions{}
	opts.QPS = 10
	opts.Init(baseURL)
	_, err := RunHTTPTest(&opts)
	if err == nil {
		t.Fatal("Expecting an error but didn't get it when connecting to bad server")
	}
	log.Infof("Got expected error from mismatch/bad server: %v", err)
}

// need to be the last test as it installs Serve() which would make
// the error test for / url above fail:

func TestServe(t *testing.T) {
	_, addr := ServeTCP("0", "/debugx1")
	port := addr.Port
	log.Infof("On addr %s found port: %d", addr, port)
	url := fmt.Sprintf("http://localhost:%d/debugx1?env=dump", port)
	if port == 0 {
		t.Errorf("outport: %d must be different", port)
	}
	time.Sleep(100 * time.Millisecond)
	o := NewHTTPOptions(url)
	o.AddAndValidateExtraHeader("X-Header: value1")
	o.AddAndValidateExtraHeader("X-Header: value2")
	code, data, _ := NewClient(o).Fetch()
	if code != http.StatusOK {
		t.Errorf("Unexpected non 200 ret code for debug url %s : %d", url, code)
	}
	if len(data) <= 100 {
		t.Errorf("Unexpected short data for debug url %s : %s", url, DebugSummary(data, 101))
	}
	if !strings.Contains(string(data), "X-Header: value1,value2") {
		t.Errorf("Multi header not found in %s", DebugSummary(data, 1024))
	}
}

func TestAbortOn(t *testing.T) {
	mux, addr := DynamicHTTPServer(false)
	mux.HandleFunc("/foo/", EchoHandler)
	baseURL := fmt.Sprintf("http://localhost:%d/", addr.Port)
	o := HTTPRunnerOptions{}
	o.URL = baseURL
	o.AbortOn = 404
	o.Exactly = 40
	o.NumThreads = 4
	o.QPS = 10
	r, err := RunHTTPTest(&o)
	if err != nil {
		t.Errorf("Error while starting runner1: %v", err)
	}
	count := r.Result().DurationHistogram.Count
	if count > int64(o.NumThreads) {
		t.Errorf("Abort1 not working, did %d requests expecting ideally 1 and <= %d", count, o.NumThreads)
	}
	o.URL += "foo/"
	r, err = RunHTTPTest(&o)
	if err != nil {
		t.Errorf("Error while starting runner2: %v", err)
	}
	count = r.Result().DurationHistogram.Count
	if count != o.Exactly {
		t.Errorf("Did %d requests when expecting all %d (non matching AbortOn)", count, o.Exactly)
	}
	o.AbortOn = 200
	r, err = RunHTTPTest(&o)
	if err != nil {
		t.Errorf("Error while starting runner3: %v", err)
	}
	count = r.Result().DurationHistogram.Count
	if count > int64(o.NumThreads) {
		t.Errorf("Abort2 not working, did %d requests expecting ideally 1 and <= %d", count, o.NumThreads)
	}
}
