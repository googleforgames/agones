// Copyright 2017 Istio Authors
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

package fhttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"fortio.org/fortio/fnet"
	"fortio.org/fortio/log"
)

func init() {
	log.SetLogLevel(log.Debug)
}

func TestGetHeaders(t *testing.T) {
	o := &HTTPOptions{}
	o.AddAndValidateExtraHeader("FOo:baR")
	oo := *o // check that copying works
	h := oo.AllHeaders()
	if len(h) != 2 { // 1 above + user-agent
		t.Errorf("Header count mismatch, got %d instead of 3", len(h))
	}
	if h.Get("Foo") != "baR" {
		t.Errorf("Foo header mismatch, got '%v'", h.Get("Foo"))
	}
	if h.Get("Host") != "" {
		t.Errorf("Host header should be nil initially, got '%v'", h.Get("Host"))
	}
	o.AddAndValidateExtraHeader("hoSt:   aBc:123")
	h = o.AllHeaders()
	if h.Get("Host") != "aBc:123" {
		t.Errorf("Host header mismatch, got '%v'", h.Get("Host"))
	}
	if len(h) != 3 { // 2 above + user-agent
		t.Errorf("Header count mismatch, got %d instead of 3", len(h))
	}
	err := o.AddAndValidateExtraHeader("foo") // missing : value
	if err == nil {
		t.Errorf("Expected error for header without value, did not get one")
	}
	o.ResetHeaders()
	h = o.AllHeaders()
	if h.Get("Host") != "" {
		t.Errorf("After reset Host header should be nil, got '%v'", h.Get("Host"))
	}
	if len(h) != 0 {
		t.Errorf("Header count mismatch after reset, got %d instead of 1", len(h))
	}
}

func TestNewHTTPRequest(t *testing.T) {
	var tests = []struct {
		url string // input
		ok  bool   // ok/error
	}{
		{"http://www.google.com/", true},
		{"ht tp://www.google.com/", false},
	}
	for _, tst := range tests {
		o := NewHTTPOptions(tst.url)
		o.AddAndValidateExtraHeader("Host: www.google.com")
		r := newHTTPRequest(o)
		if tst.ok != (r != nil) {
			t.Errorf("Got %v, expecting ok %v for url '%s'", r, tst.ok, tst.url)
		}
	}
}

func TestMultiInitAndEscape(t *testing.T) {
	// 2 escaped already
	o := NewHTTPOptions("localhost%3A8080/?delay=10ms:10,0.5s:15%25,0.25s:5")
	// shouldn't perform any escapes
	expected := "http://localhost%3A8080/?delay=10ms:10,0.5s:15%25,0.25s:5"
	if o.URL != expected {
		t.Errorf("Got initially '%s', expected '%s'", o.URL, expected)
	}
	o.AddAndValidateExtraHeader("FoO: BaR")
	// re init should not erase headers
	o.Init(o.URL)
	if o.AllHeaders().Get("Foo") != "BaR" {
		t.Errorf("Lost header after Init %+v", o.AllHeaders())
	}
}

func TestSchemeCheck(t *testing.T) {
	var tests = []struct {
		input  string
		output string
		stdcli bool
	}{
		{"https://www.google.com/", "https://www.google.com/", true},
		{"www.google.com", "http://www.google.com", false},
		{"hTTps://foo.bar:123/ab/cd", "hTTps://foo.bar:123/ab/cd", true}, // not double http:
		{"HTTP://foo.bar:124/ab/cd", "HTTP://foo.bar:124/ab/cd", false},  // not double http:
		{"", "", false},                      // and error in the logs
		{"x", "http://x", false},             //should not crash because url is shorter than prefix
		{"http:/", "http://http:/", false},   //boundary
		{"http://", "http://", false},        //boundary
		{"https://", "https://", true},       //boundary
		{"https:/", "http://https:/", false}, //boundary
	}
	for _, tst := range tests {
		o := NewHTTPOptions(tst.input)
		if o.URL != tst.output {
			t.Errorf("Got %v, expecting %v for url '%s'", o.URL, tst.output, tst.input)
		}
		if o.DisableFastClient != tst.stdcli {
			t.Errorf("Got %v, expecting %v for stdclient for url '%s'", o.DisableFastClient, tst.stdcli, tst.input)
		}
	}
}

func TestFoldFind1(t *testing.T) {
	var tests = []struct {
		haystack string // input
		needle   string // input
		found    bool   // expected result
		offset   int    // where
	}{
		{"", "", true, 0},
		{"", "A", false, -1},
		{"abc", "", true, 0},
		{"abc", "ABCD", false, -1},
		{"abc", "ABC", true, 0},
		{"aBcd", "ABC", true, 0},
		{"xaBc", "ABC", true, 1},
		{"XYZaBcUVW", "Abc", true, 3},
		{"xaBcd", "ABC", true, 1},
		{"Xa", "A", true, 1},
		{"axabaBcd", "ABC", true, 4},
		{"axabxaBcd", "ABC", true, 5},
		{"axabxaBd", "ABC", false, -1},
		{"AAAAB", "AAAB", true, 1},
		{"xAAAxAAA", "AAAB", false, -1},
		{"xxxxAc", "AB", false, -1},
		{"X-: X", "-: ", true, 1},
		{"\nX", "*X", false, -1}, // \n shouldn't fold into *
		{"*X", "\nX", false, -1}, // \n shouldn't fold into *
		{"\rX", "-X", false, -1}, // \r shouldn't fold into -
		{"-X", "\rX", false, -1}, // \r shouldn't fold into -
		{"foo\r\nContent-Length: 34\r\n", "CONTENT-LENGTH:", true, 5},
	}
	for _, tst := range tests {
		f, o := FoldFind([]byte(tst.haystack), []byte(tst.needle))
		if tst.found != f {
			t.Errorf("Got %v, expecting found %v for FoldFind('%s', '%s')", f, tst.found, tst.haystack, tst.needle)
		}
		if tst.offset != o {
			t.Errorf("Offset %d, expecting %d for FoldFind('%s', '%s')", o, tst.offset, tst.haystack, tst.needle)
		}
	}
}

func TestFoldFind2(t *testing.T) {
	var haystack [1]byte
	var needle [1]byte
	// we don't mind for these to map to eachother in exchange for 30% perf gain
	okExceptions := "@[\\]^_`{|}~"
	for i := 0; i < 127; i++ { // skipping 127 too, matches _
		haystack[0] = byte(i)
		for j := 0; j < 128; j++ {
			needle[0] = byte(j)
			sh := string(haystack[:])
			sn := string(needle[:])
			f, o := FoldFind(haystack[:], needle[:])
			shouldFind := strings.EqualFold(sh, sn)
			if i == j || shouldFind {
				if !f || o != 0 {
					t.Errorf("Not found when should: %d 0x%x '%s' matching %d 0x%x '%s'",
						i, i, sh, j, j, sn)
				}
				continue
			}
			if f || o != -1 {
				if strings.Contains(okExceptions, sh) {
					continue
				}
				t.Errorf("Found when shouldn't: %d 0x%x '%s' matching %d 0x%x '%s'",
					i, i, sh, j, j, sn)
			}
		}
	}
}

var utf8Str = "世界aBcdefGHiJklmnopqrstuvwxyZ"

func TestASCIIToUpper(t *testing.T) {
	log.SetLogLevel(log.Debug)
	var tests = []struct {
		input    string // input
		expected string // output
	}{
		{"", ""},
		{"A", "A"},
		{"aBC", "ABC"},
		{"AbC", "ABC"},
		{utf8Str, "\026LABCDEFGHIJKLMNOPQRSTUVWXYZ" /* got mangled but only first 2 */},
	}
	for _, tst := range tests {
		actual := ASCIIToUpper(tst.input)
		if tst.expected != string(actual) {
			t.Errorf("Got '%+v', expecting '%+v' for ASCIIFold('%s')", actual, tst.expected, tst.input)
		}
	}
	utf8bytes := []byte(utf8Str)
	if len(utf8bytes) != 26+6 {
		t.Errorf("Got %d utf8 bytes, expecting 6+26 for '%s'", len(utf8bytes), utf8Str)
	}
	folded := ASCIIToUpper(utf8Str)
	if len(folded) != 26+2 {
		t.Errorf("Got %d folded bytes, expecting 2+26 for '%s'", len(folded), utf8Str)
	}
}

func TestParseDecimal(t *testing.T) {
	var tests = []struct {
		input    string // input
		expected int    // output
	}{
		{"", -1},
		{"3", 3},
		{" 456cxzc", 456},
		{"-45", -1}, // - is not expected, positive numbers only
		{"3.2", 3},  // stops at first non digit
		{"    1 2", 1},
		{"0", 0},
	}
	for _, tst := range tests {
		actual := ParseDecimal([]byte(tst.input))
		if tst.expected != actual {
			t.Errorf("Got %d, expecting %d for ParseDecimal('%s')", actual, tst.expected, tst.input)
		}
	}
}

func TestParseChunkSize(t *testing.T) {
	var tests = []struct {
		input     string // input
		expOffset int    // expected offset
		expValue  int    // expected value
	}{
		// Errors :
		{"", 0, -1},
		{"0", 1, -1},
		{"0\r", 2, -1},
		{"0\n", 2, -1},
		{"g\r\n", 0, -1},
		{"0\r0\n", 4, -1},
		// Ok: (size of input is the expected offset)
		{"0\r\n", 3, 0},
		{"0x\r\n", 4, 0},
		{"f\r\n", 3, 15},
		{"10\r\n", 4, 16},
		{"fF\r\n", 4, 255},
		{"abcdef\r\n", 8, 0xabcdef},
		{"100; foo bar\r\nanother line\r\n", 14 /* and not the whole thing */, 256},
	}
	for _, tst := range tests {
		actOffset, actVal := ParseChunkSize([]byte(tst.input))
		if tst.expValue != actVal {
			t.Errorf("Got %d, expecting %d for value of ParseChunkSize('%+s')", actVal, tst.expValue, tst.input)
		}
		if tst.expOffset != actOffset {
			t.Errorf("Got %d, expecting %d for offset of ParseChunkSize('%+s')", actOffset, tst.expOffset, tst.input)
		}
	}
}

func TestDebugSummary(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"12345678", "12345678"},
		{"123456789", "123456789"},
		{"1234567890", "1234567890"},
		{"12345678901", "12345678901"},
		{"123456789012", "12: 1234...9012"},
		{"1234567890123", "13: 1234...0123"},
		{"12345678901234", "14: 1234...1234"},
		{"A\r\000\001\x80\nB", `A\r\x00\x01\x80\nB`},                   // escaping
		{"A\r\000Xyyyyyyyyy\001\x80\nB", `17: A\r\x00X...\x01\x80\nB`}, // escaping
	}
	for _, tst := range tests {
		if actual := DebugSummary([]byte(tst.input), 8); actual != tst.expected {
			t.Errorf("Got '%s', expected '%s' for DebugSummary(%q)", actual, tst.expected, tst.input)
		}
	}
}

func TestParseStatus(t *testing.T) {
	var tests = []struct {
		input    string
		expected int
	}{
		// Error cases
		{"x", 400},
		{"1::", 400},
		{"x:10", 400},
		{"555:-1", 400},
		{"555:101", 400},
		{"551:45,551:56", 400},
		// Good cases
		{"555", 555},
		{"555:100", 555},
		{"555:100%", 555},
		{"555:0", 200},
		{"555:0%", 200},
		{"551:45,551:55", 551},
		{"551:45%,551:55%", 551},
	}
	for _, tst := range tests {
		if actual := generateStatus(tst.input); actual != tst.expected {
			t.Errorf("Got %d, expected %d for generateStatus(%q)", actual, tst.expected, tst.input)
		}
	}
}

func TestParseDelay(t *testing.T) {
	var tests = []struct {
		input    string
		expected time.Duration
	}{
		// Error cases
		{"", -1},
		{"x", -1},
		{"1::", -1},
		{"x:10", -1},
		{"10ms:-1", -1},
		{"20ms:101", -1},
		{"20ms:101%", -1},
		{"10ms:45,100ms:56", -1},
		// Max delay case:
		{"10s:45,10s:55", MaxDelay},
		// Good cases
		{"100ms", 100 * time.Millisecond},
		{"100ms:100", 100 * time.Millisecond},
		{"100ms:100%", 100 * time.Millisecond},
		{"100ms:0", 0},
		{"100ms:0%", 0},
		{"10ms:45,10ms:55", 10 * time.Millisecond},
		{"10ms:45%,10ms:55%", 10 * time.Millisecond},
	}
	for _, tst := range tests {
		if actual := generateDelay(tst.input); actual != tst.expected {
			t.Errorf("Got %d, expected %d for generateStatus(%q)", actual, tst.expected, tst.input)
		}
	}
}

func TestGenerateStatusBasic(t *testing.T) {
	var tests = []struct {
		input    string
		expected int
	}{
		// Error cases
		{"x", 400},
		{"1::", 400},
		{"x:10", 400},
		{"555:x", 400},
		{"555:-1", 400},
		{"555:101", 400},
		{"551:45,551:56", 400},
		// Good cases
		{"555", 555},
		{"555:100", 555},
		{"555:0", 200},
		{"551:45,551:55", 551},
	}
	for _, tst := range tests {
		if actual := generateStatus(tst.input); actual != tst.expected {
			t.Errorf("Got %d, expected %d for generateStatus(%q)", actual, tst.expected, tst.input)
		}
	}
}

func TestGenerateStatusEdgeSum(t *testing.T) {
	st := "503:99.0,503:1.00001"
	// Gets 400 without rounding as it exceeds 100, another corner case is if you
	// add 0.1 1000 times you get 0.99999... so you may get stray 200s without Rounding
	if actual := generateStatus(st); actual != 503 {
		t.Errorf("Got %d for generateStatus(%q)", actual, st)
	}
	st += ",500:0.0001"
	if actual := generateStatus(st); actual != 400 {
		t.Errorf("Got %d for long generateStatus(%q) when expecting 400 for > 100", actual, st)
	}
}

// Round down to the nearest thousand
func roundthousand(x int) int {
	return int(float64(x)+500.) / 1000
}

func TestGenerateStatusDistribution(t *testing.T) {
	log.SetLogLevel(log.Info)
	str := "501:20,502:30,503:0.5"
	m := make(map[int]int)
	for i := 0; i < 10000; i++ {
		m[generateStatus(str)]++
	}
	if len(m) != 4 {
		t.Errorf("Unexpected result, expecting 4 statuses, got %+v", m)
	}
	if m[200]+m[501]+m[502]+m[503] != 10000 {
		t.Errorf("Unexpected result, expecting 4 statuses summing to 10000 got %+v", m)
	}
	if m[503] <= 10 {
		t.Errorf("Unexpected result, expecting at least 10 count for 0.5%% probability over 10000 got %+v", m)
	}
	// Round the data
	f01 := roundthousand(m[501]) // 20% -> 2
	f02 := roundthousand(m[502]) // 30% -> 3
	fok := roundthousand(m[200]) // rest is 50% -> 5
	f03 := roundthousand(m[503]) // 0.5% -> rounds down to 0 10s of %

	if f01 != 2 || f02 != 3 || fok != 5 || (f03 != 0) {
		t.Errorf("Unexpected distribution for %+v - wanted 2 3 5, got %d %d %d", m, f01, f02, fok)
	}
}

func TestRoundDuration(t *testing.T) {
	var tests = []struct {
		input    time.Duration
		expected time.Duration
	}{
		{0, 0},
		{1200 * time.Millisecond, 1200 * time.Millisecond},
		{1201 * time.Millisecond, 1200 * time.Millisecond},
		{1249 * time.Millisecond, 1200 * time.Millisecond},
		{1250 * time.Millisecond, 1300 * time.Millisecond},
		{1299 * time.Millisecond, 1300 * time.Millisecond},
	}
	for _, tst := range tests {
		if actual := RoundDuration(tst.input); actual != tst.expected {
			t.Errorf("Got %v, expected %v for RoundDuration(%v)", actual, tst.expected, tst.input)
		}
	}
}

func TestGenerateSize(t *testing.T) {
	var tests = []struct {
		input    string
		expected int
	}{
		// Error cases
		{"x", -1},
		{"1::", -1},
		{"x:10", -1},
		{"555:x", -1},
		{"555:-1", -1},
		{"555:101", -1},
		{"551:45,551:56", -1},
		// Good cases
		{"", -1},
		{"512", 512},
		{"512:100", 512},
		{"512:0", -1},
		{"512:45,512:55", 512},
		{"0", 0}, // and not -1
		{"262144", 262144},
		{"262145", fnet.MaxPayloadSize}, // MaxSize test
		{"1000000:10,2000000:90", 262144},
	}
	for _, tst := range tests {
		if actual := generateSize(tst.input); actual != tst.expected {
			t.Errorf("Got %d, expected %d for generateSize(%q)", actual, tst.expected, tst.input)
		}
	}
}

func TestPayloadWithEchoBack(t *testing.T) {
	var tests = []struct {
		payload           []byte
		disableFastClient bool
	}{
		{[]byte{44, 45, 00, 46, 47}, false},
		{[]byte{44, 45, 00, 46, 47}, true},
		{[]byte("groß"), false},
		{[]byte("groß"), true},
	}
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	url := fmt.Sprintf("http://localhost:%d/", a.Port)
	for _, test := range tests {
		opts := NewHTTPOptions(url)
		opts.DisableFastClient = test.disableFastClient
		opts.Payload = test.payload
		cli := NewClient(opts)
		code, body, header := cli.Fetch()
		if code != 200 {
			t.Errorf("Unexpected error %d", code)
		}
		if !bytes.Equal(body[header:], test.payload) {
			t.Errorf("Got %s, expected %q from echo", DebugSummary(body, 512), test.payload)
		}
		if !test.disableFastClient {
			cli.Close()
		}
	}
}

// Many of the earlier http tests are through httprunner but new tests should go here

func TestUnixDomainHttp(t *testing.T) {
	uds := fnet.GetUniqueUnixDomainPath("fortio-http-test-uds")
	_, addr := Serve(uds, "/debug1")
	if addr == nil {
		t.Fatalf("Error for Serve for %s", uds)
	}
	o := HTTPOptions{UnixDomainSocket: uds, URL: "http://foo.bar:123/debug1"}
	client := NewClient(&o)
	code, data, _ := client.Fetch()
	if code != http.StatusOK {
		t.Errorf("Got error %d fetching uds %s", code, uds)
	}
	if !strings.Contains(string(data), "Host: foo.bar:123") {
		t.Errorf("Didn't find expected Host header in debug handler: %s", DebugSummary(data, 1024))
	}
}

func TestEchoBack(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	v := url.Values{}
	v.Add("foo", "bar")
	url := fmt.Sprintf("http://localhost:%d/?delay=2s", a.Port) // trigger max delay
	resp, err := http.PostForm(url, v)
	if err != nil {
		t.Fatalf("post form err %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("readall err %v", err)
	}
	expected := "foo=bar"
	if string(b) != expected {
		t.Errorf("Got %s while expected %s", DebugSummary(b, 128), expected)
	}
}

func TestH10Cli(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	url := fmt.Sprintf("http://localhost:%d/", a.Port)
	opts := NewHTTPOptions(url)
	opts.HTTP10 = true
	opts.AddAndValidateExtraHeader("Host: mhostname")
	cli := NewFastClient(opts)
	code, _, _ := cli.Fetch()
	if code != 200 {
		t.Errorf("http 1.0 unexpected error %d", code)
	}
	s := cli.(*FastClient).socket
	if s != nil {
		t.Errorf("http 1.0 socket should be nil after fetch (no keepalive) %+v instead", s)
	}
	cli.Close()
}

func TestSmallBufferAndNoKeepAlive(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	BufferSizeKb = 16
	sz := BufferSizeKb * 1024
	url := fmt.Sprintf("http://localhost:%d/?size=%d", a.Port, sz+1) // trigger buffer problem
	opts := NewHTTPOptions(url)
	cli := NewFastClient(opts)
	_, data, _ := cli.Fetch()
	recSz := len(data)
	if recSz > sz {
		t.Errorf("config1: was expecting truncated read, got %d", recSz)
	}
	cli.Close()
	// Same test without keepalive (exercises a different path)
	opts.DisableKeepAlive = true
	cli = NewFastClient(opts)
	_, data, _ = cli.Fetch()
	recSz = len(data)
	if recSz > sz {
		t.Errorf("config2: was expecting truncated read, got %d", recSz)
	}
	cli.Close()
}

func TestBadUrl(t *testing.T) {
	opts := NewHTTPOptions("not a valid url")
	cli := NewFastClient(opts)
	if cli != nil {
		t.Errorf("config1: got a client %v despite bogus url %s", cli, opts.URL)
		cli.Close()
	}
	opts.URL = "http://doesnotexist.istio.io"
	cli = NewFastClient(opts)
	if cli != nil {
		t.Errorf("config2: got a client %v despite bogus url %s", cli, opts.URL)
		cli.Close()
	}
}

func TestDefaultPort(t *testing.T) {
	url := "http://fortio.istio.io/" // shall imply port 80
	opts := NewHTTPOptions(url)
	cli := NewFastClient(opts)
	code, _, _ := cli.Fetch()
	if code != 303 {
		t.Errorf("unexpected code for %s: %d (expecting 303 redirect to https)", url, code)
	}
	conn := cli.(*FastClient).connect()
	if conn != nil {
		p := conn.RemoteAddr().(*net.TCPAddr).Port
		if p != 80 {
			t.Errorf("unexpected port for %s: %d", url, p)
		}
		conn.Close()
	} else {
		t.Errorf("unable to connect to %s", url)
	}
	cli.Close()
	opts.URL = "https://fortio.istio.io" // will be https port 443
	opts.Insecure = true                 // not needed as we have valid certs but to exercise that code
	cli = NewFastClient(opts)
	if cli != nil {
		// If https support was added, remove this whitebox/for coverage purpose assertion
		t.Errorf("fast client isn't supposed to support https (yet), got %v", cli)
	}
	cli = NewClient(opts)
	if cli == nil {
		t.Fatalf("Couldn't get a client using NewClient on modified opts.")
	}
	// currently fast client fails with https:
	code, _, _ = cli.Fetch()
	if code != 200 {
		t.Errorf("Standard client http error code %d", code)
	}
	cli.Close()
}

// Test for bug #127

var testBody = "delayedChunkedSize-body"

func delayedChunkedSize(w http.ResponseWriter, r *http.Request) {
	log.LogVf("delayedChunkedSize %v %v %v %v", r.Method, r.URL, r.Proto, r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	flusher.Flush()
	time.Sleep(1 * time.Second)
	w.Write([]byte(testBody))
}

func TestNoFirstChunkSizeInitially(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", delayedChunkedSize)
	url := fmt.Sprintf("http://localhost:%d/delayedChunkedSize", a.Port)
	o := HTTPOptions{URL: url}
	client := NewClient(&o)
	code, data, header := client.Fetch() // used to panic/bug #127
	t.Logf("delayedChunkedSize result code %d, data len %d, headerlen %d", code, len(data), header)
	if code != 200 {
		t.Errorf("Got %d instead of 200", code)
	}
	expected := "17\r\n" + testBody + "\r\n0\r\n\r\n" // 17 is hex size of testBody
	if string(data[header:]) != expected {
		t.Errorf("Got %s not as expected %q at offset %d", DebugSummary(data, 256), expected, header)
	}
}

func TestInvalidRequest(t *testing.T) {
	o := HTTPOptions{
		URL:            "http://www.google.com/", // valid url
		NumConnections: -3,                       // bogus NumConnections will get fixed
		HTTPReqTimeOut: -1,
	}
	client := NewStdClient(&o)
	if o.NumConnections <= 0 {
		t.Errorf("Got %d NumConnections, was expecting normalization to 1", o.NumConnections)
	}
	client.ChangeURL(" http://bad.url.with.space.com/") // invalid url
	// should not crash (issue #93), should error out
	code, _, _ := client.Fetch()
	if code != http.StatusBadRequest {
		t.Errorf("Got %d code while expecting bad request (%d)", code, http.StatusBadRequest)
	}
	o.URL = client.url
	c2 := NewStdClient(&o)
	if c2 != nil {
		t.Errorf("Got non nil client %+v code while expecting nil for bad request", c2)
	}
}

func TestPayloadSizeSmall(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	for _, size := range []int{768, 0, 1} {
		url := fmt.Sprintf("http://localhost:%d/with-size?size=%d", a.Port, size)
		o := HTTPOptions{URL: url}
		client := NewClient(&o)
		code, data, header := client.Fetch() // used to panic/bug #127
		t.Logf("TestPayloadSize result code %d, data len %d, headerlen %d", code, len(data), header)
		if code != http.StatusOK {
			t.Errorf("Got %d instead of 200", code)
		}
		if len(data)-header != size {
			t.Errorf("Got len(data)-header %d not as expected %d : got %s", len(data)-header, size, DebugSummary(data, 512))
		}
	}
}

// TODO: improve/unify/simplify those payload/POST tests: just go to /debug handler for both clients and check what is echoed back

func TestPayloadForClient(t *testing.T) {
	var tests = []struct {
		contentType    string
		payload        []byte
		expectedMethod string
	}{
		{"application/json",
			[]byte("{\"test\" : \"test\"}"),
			"POST"},
		{"application/xml",
			[]byte("<test test=\"test\">"),
			"POST"},
		{"",
			nil,
			"GET"},
	}
	for _, test := range tests {
		hOptions := HTTPOptions{}
		hOptions.URL = "www.google.com"
		hOptions.ContentType = test.contentType
		hOptions.Payload = test.payload
		client := NewStdClient(&hOptions)
		contentType := client.req.Header.Get("Content-Type")
		if contentType != test.contentType {
			t.Errorf("Got %s, expected %s as a content type", contentType, test.contentType)
		}
		method := client.req.Method
		if method != test.expectedMethod {
			t.Errorf("Got %s, expected %s as a method", method, test.expectedMethod)
		}
		body := client.req.Body
		if body == nil {
			if len(test.payload) > 0 {
				t.Errorf("Got empty nil body, expected %s as a body", test.payload)
			}
			continue
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(body)
		payload := buf.Bytes()
		if !bytes.Equal(payload, test.payload) {
			t.Errorf("Got %s, expected %s as a body", string(payload), string(test.payload))
		}
	}
}

func TestPayloadForFastClient(t *testing.T) {
	var tests = []struct {
		contentType     string
		payload         []byte
		expectedReqBody string
	}{
		{"application/json",
			[]byte("{\"test\" : \"test\"}"),
			fmt.Sprintf("POST / HTTP/1.1\r\nHost: www.google.com\r\nContent-Length: 17\r\nContent-Type: "+
				"application/json\r\nUser-Agent: %s\r\n\r\n{\"test\" : \"test\"}", userAgent)},
		{"application/xml",
			[]byte("<test test=\"test\">"),
			fmt.Sprintf("POST / HTTP/1.1\r\nHost: www.google.com\r\nContent-Length: 18\r\nContent-Type: "+
				"application/xml\r\nUser-Agent: %s\r\n\r\n<test test=\"test\">", userAgent)},
		{"",
			nil,
			fmt.Sprintf("GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: %s\r\n\r\n", userAgent)},
	}
	for _, test := range tests {
		hOptions := HTTPOptions{}
		hOptions.URL = "www.google.com"
		hOptions.ContentType = test.contentType
		hOptions.Payload = test.payload
		client := NewFastClient(&hOptions)
		body := string(client.(*FastClient).req)
		if body != test.expectedReqBody {
			t.Errorf("Got\n%s\nexpecting\n%s", body, test.expectedReqBody)
		}
	}
}

func TestPayloadSizeLarge(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/", EchoHandler)
	//basic client 128k buffer can't do 200k, also errors out on non 200 codes so doing this other bg
	size := 200000
	url := fmt.Sprintf("http://localhost:%d/with-size?size=%d&status=888", a.Port, size)
	o := HTTPOptions{URL: url, DisableFastClient: true}
	client := NewClient(&o)
	code, data, header := client.Fetch() // used to panic/bug #127
	t.Logf("TestPayloadSize result code %d, data len %d, headerlen %d", code, len(data), header)
	if code != 888 {
		t.Errorf("Got %d instead of 888", code)
	}
	if len(data)-header != size {
		t.Errorf("Got len(data)-header %d not as expected %d : got %s", len(data)-header, size, DebugSummary(data, 512))
	}
}

func TestDebugHandlerSortedHeaders(t *testing.T) {
	m, a := DynamicHTTPServer(false)
	m.HandleFunc("/debug", DebugHandler)
	url := fmt.Sprintf("http://localhost:%d/debug", a.Port)
	o := HTTPOptions{URL: url, DisableFastClient: true}
	o.AddAndValidateExtraHeader("BBB: bbb")
	o.AddAndValidateExtraHeader("CCC: ccc")
	o.AddAndValidateExtraHeader("ZZZ: zzz")
	o.AddAndValidateExtraHeader("AAA: aaa")
	client := NewClient(&o)
	code, data, header := client.Fetch() // used to panic/bug #127
	t.Logf("TestDebugHandlerSortedHeaders result code %d, data len %d, headerlen %d", code, len(data), header)
	if code != http.StatusOK {
		t.Errorf("Got %d instead of 200", code)
	}
	//remove the first line ('Φορτίο version...') from the body
	body := string(data)
	i := strings.Index(body, "\n")
	body = body[i+1:]
	expected := fmt.Sprintf("\nGET /debug HTTP/1.1\n\n"+
		"headers:\n\n"+
		"Host: localhost:%d\n"+
		"Aaa: aaa\n"+
		"Bbb: bbb\n"+
		"Ccc: ccc\n"+
		"User-Agent: %s\n"+
		"Zzz: zzz\n\n"+
		"body:\n\n\n", a.Port, userAgent)
	if body != expected {
		t.Errorf("Get body: %s not as expected: %s", body, expected)
	}
}

func TestEchoHeaders(t *testing.T) {
	_, a := ServeTCP("0", "")
	var headers = []struct {
		key   string
		value string
	}{
		{"Foo", "Bar1"},
		{"Foo", "Bar2"}, // Test multiple same header
		{"X", "Y"},
		{"Z", "abc def:xyz"},
	}
	v := url.Values{}
	for _, pair := range headers {
		v.Add("header", pair.key+":"+pair.value)
	}
	// minimal manual encoding (only escape the space) + errors for coverage sake
	var urls []string
	urls = append(urls,
		fmt.Sprintf("http://localhost:%d/echo?size=10&header=Foo:Bar1&header=Foo:Bar2&header=X:Y&header=Z:abc+def:xyz&header=&header=Foo",
			a.Port))
	// proper encoding
	urls = append(urls, fmt.Sprintf("http://localhost:%d/echo?%s", a.Port, v.Encode()))
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed get for %s : %v", url, err)
		}
		t.Logf("TestEchoHeaders url = %s : status %s", url, resp.Status)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Got %d instead of 200", resp.StatusCode)
		}
		for _, pair := range headers {
			got := resp.Header[pair.key]
			found := false
			for _, v := range got {
				if v == pair.value {
					found = true
					break // found == good
				}
			}
			if !found {
				t.Errorf("Mismatch: got %+v and didn't find \"%s\" for header %s (url %s)", got, pair.value, pair.key, url)
			}
		}
	}
}

func TestPPROF(t *testing.T) {
	mux, addrN := HTTPServer("test pprof", "0")
	addr := addrN.(*net.TCPAddr)
	url := fmt.Sprintf("localhost:%d/debug/pprof/heap?debug=1", addr.Port)
	code, _ := Fetch(&HTTPOptions{URL: url})
	if code != http.StatusNotFound {
		t.Errorf("Got %d instead of expected 404/not found for %s", code, url)
	}
	SetupPPROF(mux)
	code, data := FetchURL(url)
	if code != http.StatusOK {
		t.Errorf("Got %d %s instead of ok for %s", code, DebugSummary(data, 256), url)
	}
	if !bytes.Contains(data, []byte("TotalAlloc")) {
		t.Errorf("Result %s doesn't contain expected TotalAlloc", DebugSummary(data, 1024))
	}
}

func TestFetchAndOnBehalfOf(t *testing.T) {
	mux, addr := ServeTCP("0", "/debug")
	mux.Handle("/fetch/", http.StripPrefix("/fetch/", http.HandlerFunc(FetcherHandler)))
	url := fmt.Sprintf("localhost:%d/fetch/localhost:%d/debug", addr.Port, addr.Port)
	code, data := Fetch(&HTTPOptions{URL: url})
	if code != http.StatusOK {
		t.Errorf("Got %d %s instead of ok for %s", code, DebugSummary(data, 256), url)
	}
	// ideally we'd check more of the header but it can be 127.0.0.1:port or [::1]:port depending on ipv6 support etc...
	if !bytes.Contains(data, []byte("X-On-Behalf-Of: ")) {
		t.Errorf("Result %s doesn't contain expected On-Behalf-Of:", DebugSummary(data, 1024))
	}
}

func TestServeError(t *testing.T) {
	_, addr := Serve("0", "")
	port := fnet.GetPort(addr)
	mux2, addr2 := Serve(port, "")
	if mux2 != nil || addr2 != nil {
		t.Errorf("2nd Serve() on same port %v should have failed: %v %v", port, mux2, addr2)
	}
}

func testCacheHeaderHandler(w http.ResponseWriter, r *http.Request) {
	LogRequest(r, "testCacheHeader")
	CacheOn(w)
	w.Write([]byte("cache me"))
}

func TestCache(t *testing.T) {
	mux, addr := ServeTCP("0", "")
	mux.HandleFunc("/cached", testCacheHeaderHandler)
	baseURL := fmt.Sprintf("http://localhost:%d/", addr.Port)
	o := NewHTTPOptions(baseURL)
	code, data := Fetch(o)
	if code != 200 {
		t.Errorf("error fetching %s: %v %s", o.URL, code, DebugSummary(data, 256))
	}
	expectedWithCache := []byte("Cache-Control:")
	if bytes.Contains(data, expectedWithCache) {
		t.Errorf("Got %s when shouldn't have for %s: %v", expectedWithCache, o.URL, DebugSummary(data, 256))
	}
	o.URL += "cached"
	code, data = Fetch(o)
	if code != 200 {
		t.Errorf("error fetching %s: %v %s", o.URL, code, DebugSummary(data, 256))
	}
	if !bytes.Contains(data, expectedWithCache) {
		t.Errorf("Didn't get %s when should have for %s: %v", expectedWithCache, o.URL, DebugSummary(data, 256))
	}
}

func TestRedirector(t *testing.T) {
	addr := RedirectToHTTPS(":0")
	relativeURL := "/foo/bar?some=param&anotherone"
	port := fnet.GetPort(addr)
	url := fmt.Sprintf("http://localhost:%s%s", port, relativeURL)
	opts := NewHTTPOptions(url)
	opts.AddAndValidateExtraHeader("Host: foo.istio.io")
	code, data := Fetch(opts)
	if code != http.StatusSeeOther {
		t.Errorf("Got %d %s instead of %d for %s", code, DebugSummary(data, 256), http.StatusSeeOther, url)
	}
	if !bytes.Contains(data, []byte("Location: https://foo.istio.io"+relativeURL)) {
		t.Errorf("Result %s doesn't contain Location: redirect", DebugSummary(data, 1024))
	}
	// 2nd one should fail
	addr2 := RedirectToHTTPS(port)
	if addr2 != nil {
		t.Errorf("2nd RedirectToHTTPS() on same port %s should have failed: %v", port, addr2)

	}
}

var testNeedEscape = "<a href='http://google.com'>link</a>"

func escapeTestHandler(w http.ResponseWriter, r *http.Request) {
	LogRequest(r, "escapeTestHandler")
	out := NewHTMLEscapeWriter(w)
	fmt.Fprintln(out, testNeedEscape)
}

func TestHTMLEscapeWriter(t *testing.T) {
	mux, addr := HTTPServer("test escape", ":0")
	mux.HandleFunc("/", escapeTestHandler)
	url := fmt.Sprintf("http://localhost:%s/", fnet.GetPort(addr))
	code, data := FetchURL(url)
	if code != http.StatusOK {
		t.Errorf("Got %d %s instead of ok for %s", code, DebugSummary(data, 256), url)
	}
	if !bytes.Contains(data, []byte("&lt;a href=&#39;http://google.com&#39;&gt;link")) {
		t.Errorf("Result %s doesn't contain expected escaped html:", DebugSummary(data, 1024))
	}
}

func TestNewHTMLEscapeWriterError(t *testing.T) {
	log.Infof("Expect error complaining about not an http/flusher:")
	out := NewHTMLEscapeWriter(os.Stdout) // should cause flusher to be null
	hw := out.(*HTMLEscapeWriter)
	if hw.Flusher != nil {
		t.Errorf("Shouldn't have a flusher when not passing in an http: %+v", hw.Flusher)
	}
}

func TestDefaultHeadersAndOptionsInit(t *testing.T) {
	_, addr := ServeTCP("0", "/debug")
	// Un initialized http options:
	o := HTTPOptions{URL: fmt.Sprintf("http://localhost:%d/debug", addr.Port)}
	o1 := o
	cli1 := NewStdClient(&o1)
	code, data, _ := cli1.Fetch()
	if code != 200 {
		t.Errorf("Non ok code %d for debug default fetch1", code)
	}
	expected := []byte("User-Agent: fortio.org/fortio-")
	if !bytes.Contains(data, expected) {
		t.Errorf("Didn't find default header echoed back in std client1 %s (expecting %s)", DebugSummary(data, 512), expected)
	}
	o2 := o
	cli2 := NewFastClient(&o2)
	code, data, _ = cli2.Fetch()
	if code != 200 {
		t.Errorf("Non ok code %d for debug default fetch2", code)
	}
	if !bytes.Contains(data, expected) {
		t.Errorf("Didn't find default header echoed back in fast client1 %s (expecting %s)", DebugSummary(data, 512), expected)
	}
}

func TestAddHTTPS(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"foo", "https://foo"},
		{"foo.com", "https://foo.com"},
		{"http://foo.com", "https://foo.com"},
		{"https://foo.com", "https://foo.com"},
		{"hTTps://foo.com", "https://foo.com"},
		{"hTTp://foo.com", "https://foo.com"},
		{"http://fOO.com", "https://fOO.com"},
		{"hTTp://foo.com/about.html", "https://foo.com/about.html"},
		{"hTTp://foo.com/ABOUT.html", "https://foo.com/ABOUT.html"},
		{"hTTp://foo.com/BaR", "https://foo.com/BaR"},
		{"https://foo.com/BaR", "https://foo.com/BaR"},
		{"http", "https://http"},
	}

	for _, test := range tests {
		output := AddHTTPS(test.input)
		if output != test.expected {
			t.Errorf("%s is received but %s was expected", output, test.expected)
		}
	}
}

func TestValidateAndAddBasicAuthentication(t *testing.T) {
	var tests = []struct {
		o                  HTTPOptions
		isCredentialsValid bool
		isAuthHeaderAdded  bool
	}{
		{HTTPOptions{UserCredentials: "foo:foo"}, true, true},
		{HTTPOptions{UserCredentials: "foofoo"}, false, false},
		{HTTPOptions{UserCredentials: ""}, true, false},
	}

	for _, test := range tests {
		h := make(http.Header)
		err := test.o.ValidateAndAddBasicAuthentication(h)
		if err == nil && !test.isCredentialsValid {
			t.Errorf("Error was not expected for %s", test.o.UserCredentials)
		}
		if test.isAuthHeaderAdded && len(h.Get("Authorization")) <= 0 {
			t.Errorf("Authorization header was expected for %s credentials", test.o.UserCredentials)
		}

	}
}

// --- for bench mark/comparison

func asciiFold0(str string) []byte {
	return []byte(strings.ToUpper(str))
}

var toLowerMaskRune = rune(toUpperMask)

func toLower(r rune) rune {
	return r & toLowerMaskRune
}

func asciiFold1(str string) []byte {
	return []byte(strings.Map(toLower, str))
}

var lw []byte

func BenchmarkASCIIFoldNormalToLower(b *testing.B) {
	for n := 0; n < b.N; n++ {
		lw = asciiFold0(utf8Str)
	}
}
func BenchmarkASCIIFoldCustomToLowerMap(b *testing.B) {
	for n := 0; n < b.N; n++ {
		lw = asciiFold1(utf8Str)
	}
}

// Package's version (3x fastest)
func BenchmarkASCIIToUpper(b *testing.B) {
	log.SetLogLevel(log.Warning)
	for n := 0; n < b.N; n++ {
		lw = ASCIIToUpper(utf8Str)
	}
}

// Note: newline inserted in set-cookie line because of linter (line too long)
var testHaystack = []byte(`HTTP/1.1 200 OK
Date: Sun, 16 Jul 2017 21:00:29 GMT
Expires: -1
Cache-Control: private, max-age=0
Content-Type: text/html; charset=ISO-8859-1
P3P: CP="This is not a P3P policy! See https://www.google.com/support/accounts/answer/151657?hl=en for more info."
Server: gws
X-XSS-Protection: 1; mode=block
X-Frame-Options: SAMEORIGIN
Set-Cookie: NID=107=sne5itxJgY_4dD951psa7cyP_rQ3ju-J9p0QGmKYl0l0xUVSVmGVeX8smU0VV6FyfQnZ4kkhaZ9ozxLpUWH-77K_0W8aXzE3
PDQxwAynvJgGGA9rMRB9bperOblUOQ3XilG6B5-8auMREgbc; expires=Mon, 15-Jan-2018 21:00:29 GMT; path=/; domain=.google.com; HttpOnly
Accept-Ranges: none
Vary: Accept-Encoding
Transfer-Encoding: chunked
`)

func FoldFind0(haystack []byte, needle []byte) (bool, int) {
	offset := strings.Index(strings.ToUpper(string(haystack)), string(needle))
	found := (offset >= 0)
	return found, offset
}

// -- benchmarks --

func BenchmarkFoldFind0(b *testing.B) {
	needle := []byte("VARY")
	for n := 0; n < b.N; n++ {
		FoldFind0(testHaystack, needle)
	}
}

func BenchmarkFoldFind(b *testing.B) {
	needle := []byte("VARY")
	for n := 0; n < b.N; n++ {
		FoldFind(testHaystack, needle)
	}
}

// -- end of benchmark tests / end of this file
