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

package fnet

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"bytes"

	"fortio.org/fortio/log"
	"fortio.org/fortio/version"
)

func TestNormalizePort(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			"port number only",
			"8080",
			":8080",
		},
		{
			"IPv4 host:port",
			"10.10.10.1:8080",
			"10.10.10.1:8080",
		},
		{
			"IPv6 [host]:port",
			"[2001:db1::1]:8080",
			"[2001:db1::1]:8080",
		},
	}

	for _, tc := range tests {
		port := NormalizePort(tc.input)
		if port != tc.output {
			t.Errorf("Test case %s failed to normailze port %s\n\texpected: %s\n\t  actual: %s",
				tc.name,
				tc.input,
				tc.output,
				port,
			)
		}
	}
}

func TestListen(t *testing.T) {
	l, a := Listen("test listen1", "0")
	if l == nil || a == nil {
		t.Fatalf("Unexpected nil in Listen() %v %v", l, a)
	}
	if a.(*net.TCPAddr).Port == 0 {
		t.Errorf("Unexpected 0 port after listen %+v", a)
	}
	_ = l.Close() // nolint: gas
}

func TestListenFailure(t *testing.T) {
	_, a1 := Listen("test listen2", "0")
	if a1.(*net.TCPAddr).Port == 0 {
		t.Errorf("Unexpected 0 port after listen %+v", a1)
	}
	l, a := Listen("this should fail", GetPort(a1))
	if l != nil || a != nil {
		t.Errorf("listen that should error got %v %v instead of nil", l, a)
	}
}

func TestResolveDestination(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		want        string
	}{
		// Error cases:
		{"missing :", "foo", ""},
		{"using ip:bogussvc", "8.8.8.8:doesnotexisthopefully", ""},
		{"using bogus hostname", "doesnotexist.istio.io:443", ""},
		// Good cases:
		{"using ip:portname", "8.8.8.8:http", "8.8.8.8:80"},
		{"using ip:port", "8.8.8.8:12345", "8.8.8.8:12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDestination(tt.destination)
			gotStr := ""
			if got != nil {
				gotStr = got.String()
			}
			if gotStr != tt.want {
				t.Errorf("ResolveDestination(%s) = %v, want %s", tt.destination, got, tt.want)
			}
		})
	}
}

func TestResolveDestinationMultipleIps(t *testing.T) {
	addr := ResolveDestination("www.google.com:443")
	t.Logf("Found google addr %+v", addr)
	if addr == nil {
		t.Error("got nil address for google")
	}
}

func TestProxy(t *testing.T) {
	addr := ProxyToDestination(":0", "www.google.com:80")
	dAddr := net.TCPAddr{Port: addr.(*net.TCPAddr).Port}
	d, err := net.DialTCP("tcp", nil, &dAddr)
	if err != nil {
		t.Fatalf("can't connect to our proxy: %v", err)
	}
	defer d.Close()
	data := "HEAD / HTTP/1.0\r\nUser-Agent: fortio-unit-test-" + version.Long() + "\r\n\r\n"
	d.Write([]byte(data))
	d.CloseWrite()
	res := make([]byte, 4096)
	n, err := d.Read(res)
	if err != nil {
		t.Errorf("read error with proxy: %v", err)
	}
	resStr := string(res[:n])
	expectedStart := "HTTP/1.0 200 OK\r\n"
	if !strings.HasPrefix(resStr, expectedStart) {
		t.Errorf("Unexpected reply '%q', expected starting with '%q'", resStr, expectedStart)
	}
}

func TestBadGetUniqueUnixDomainPath(t *testing.T) {
	badPath := []byte{0x41, 0, 0x42}
	fname := GetUniqueUnixDomainPath(string(badPath))
	if fname != "/tmp/fortio-default-uds" {
		t.Errorf("Got %s when expecting default/error case for bad prefix", fname)
	}
}

func TestDefaultGetUniqueUnixDomainPath(t *testing.T) {
	n1 := GetUniqueUnixDomainPath("")
	n2 := GetUniqueUnixDomainPath("")
	if n1 == n2 {
		t.Errorf("Got %s and %s when expecting unique names", n1, n2)
	}
}

func TestUnixDomain(t *testing.T) {
	// Test through the proxy as well (which indirectly tests Listen)
	fname := GetUniqueUnixDomainPath("fortio-uds-test")
	addr := ProxyToDestination(fname, "www.google.com:80")
	defer os.Remove(fname) // to not leak the temp socket
	if addr == nil {
		t.Fatalf("Nil socket in unix socket proxy listen")
	}
	hp := NormalizeHostPort("", addr)
	expected := fmt.Sprintf("-unix-socket=%s", fname)
	if hp != expected {
		t.Errorf("Got %s, expected %s from NormalizeHostPort(%v)", hp, expected, addr)
	}
	dAddr := net.UnixAddr{Name: fname, Net: UnixDomainSocket}
	d, err := net.DialUnix(UnixDomainSocket, nil, &dAddr)
	if err != nil {
		t.Fatalf("can't connect to our proxy using unix socket %v: %v", fname, err)
	}
	defer d.Close()
	data := "HEAD / HTTP/1.0\r\nUser-Agent: fortio-unit-test-" + version.Long() + "\r\n\r\n"
	d.Write([]byte(data))
	d.CloseWrite()
	res := make([]byte, 4096)
	n, err := d.Read(res)
	if err != nil {
		t.Errorf("read error with proxy: %v", err)
	}
	resStr := string(res[:n])
	expectedStart := "HTTP/1.0 200 OK\r\n"
	if !strings.HasPrefix(resStr, expectedStart) {
		t.Errorf("Unexpected reply '%q', expected starting with '%q'", resStr, expectedStart)
	}

}
func TestProxyErrors(t *testing.T) {
	addr := ProxyToDestination(":0", "doesnotexist.istio.io:80")
	dAddr := net.TCPAddr{Port: addr.(*net.TCPAddr).Port}
	d, err := net.DialTCP("tcp", nil, &dAddr)
	if err != nil {
		t.Fatalf("can't connect to our proxy: %v", err)
	}
	defer d.Close()
	res := make([]byte, 4096)
	n, err := d.Read(res)
	if err == nil {
		t.Errorf("didn't get expected error with proxy %d", n)
	}
	// 2nd proxy on same port should fail
	addr2 := ProxyToDestination(GetPort(addr), "www.google.com:80")
	if addr2 != nil {
		t.Errorf("Second proxy on same port should have failed, got %+v", addr2)
	}
}
func TestResolveIpV6(t *testing.T) {
	addr := Resolve("[::1]", "http")
	addrStr := addr.String()
	expected := "[::1]:80"
	if addrStr != expected {
		t.Errorf("Got '%s' instead of '%s'", addrStr, expected)
	}
}

func TestJoinHostAndPort(t *testing.T) {
	var tests = []struct {
		inputPort string
		addr      *net.TCPAddr
		expected  string
	}{
		{"8080", &net.TCPAddr{
			IP:   []byte{192, 168, 2, 3},
			Port: 8081,
		}, "localhost:8081"},
		{"192.168.30.14:8081", &net.TCPAddr{
			IP:   []byte{192, 168, 30, 15},
			Port: 8080,
		}, "192.168.30.15:8080"},
		{":8080",
			&net.TCPAddr{
				IP:   []byte{0, 0, 0, 1},
				Port: 8080,
			},
			"localhost:8080"},
		{"",
			&net.TCPAddr{
				IP:   []byte{192, 168, 30, 14},
				Port: 9090,
			}, "localhost:9090"},
		{"http",
			&net.TCPAddr{
				IP:   []byte{192, 168, 30, 14},
				Port: 9090,
			}, "localhost:9090"},
		{"192.168.30.14:9090",
			&net.TCPAddr{
				IP:   []byte{192, 168, 30, 14},
				Port: 9090,
			}, "192.168.30.14:9090"},
	}
	for _, test := range tests {
		urlHostPort := NormalizeHostPort(test.inputPort, test.addr)
		if urlHostPort != test.expected {
			t.Errorf("%s is received  but %s was expected", urlHostPort, test.expected)
		}
	}
}

func TestChangeMaxPayloadSize(t *testing.T) {
	var tests = []struct {
		input    int
		expected int
	}{
		// negative test cases
		{-1, 0},
		// lesser than current default
		{0, 0},
		{64, 64},
		// Greater than current default
		{987 * 1024, 987 * 1024},
	}
	for _, tst := range tests {
		ChangeMaxPayloadSize(tst.input)
		actual := len(Payload)
		if len(Payload) != tst.expected {
			t.Errorf("Got %d, expected %d for ChangeMaxPayloadSize(%d)", actual, tst.expected, tst.input)
		}
	}
}

func TestValidatePayloadSize(t *testing.T) {
	ChangeMaxPayloadSize(256 * 1024)
	var tests = []struct {
		input    int
		expected int
	}{
		{257 * 1024, MaxPayloadSize},
		{10, 10},
		{0, 0},
		{-1, 0},
	}
	for _, test := range tests {
		size := test.input
		ValidatePayloadSize(&size)
		if size != test.expected {
			t.Errorf("Got %d, expected %d for ValidatePayloadSize(%d)", size, test.expected, test.input)
		}
	}
}

func TestGenerateRandomPayload(t *testing.T) {
	ChangeMaxPayloadSize(256 * 1024)
	var tests = []struct {
		input    int
		expected int
	}{
		{257 * 1024, MaxPayloadSize},
		{10, 10},
		{0, 0},
		{-1, 0},
	}
	for _, test := range tests {
		text := GenerateRandomPayload(test.input)
		if len(text) != test.expected {
			t.Errorf("Got %d, expected %d for GenerateRandomPayload(%d) payload size", len(text), test.expected, test.input)
		}
	}
}

func TestReadFileForPayload(t *testing.T) {
	var tests = []struct {
		payloadFile  string
		expectedText []byte
	}{
		{payloadFile: "../.testdata/payloadTest1.txt", expectedText: []byte("{\"test\":\"test\"}")},
		{payloadFile: "", expectedText: nil},
	}

	for _, test := range tests {
		data, err := ReadFileForPayload(test.payloadFile)
		if err != nil && len(test.expectedText) > 0 {
			t.Errorf("Error should not be happened for ReadFileForPayload")
		}
		if !bytes.Equal(data, test.expectedText) {
			t.Errorf("Got %s, expected %s for ReadFileForPayload()", string(data), string(test.expectedText))
		}
	}
}

func TestGeneratePayload(t *testing.T) {
	var tests = []struct {
		payloadFile    string
		payloadSize    int
		payload        string
		expectedResLen int
	}{
		{payloadFile: "../.testdata/payloadTest1.txt", payloadSize: 123, payload: "",
			expectedResLen: len("{\"test\":\"test\"}")},
		{payloadFile: "nottestmock", payloadSize: 0, payload: "{\"test\":\"test1\"}",
			expectedResLen: 0},
		{payloadFile: "", payloadSize: 123, payload: "{\"test\":\"test1\"}",
			expectedResLen: 123},
		{payloadFile: "", payloadSize: 0, payload: "{\"test\":\"test1\"}",
			expectedResLen: len("{\"test\":\"test1\"}")},
		{payloadFile: "", payloadSize: 0, payload: "",
			expectedResLen: 0},
	}

	for _, test := range tests {
		payload := GeneratePayload(test.payloadFile, test.payloadSize, test.payload)
		if len(payload) != test.expectedResLen {
			t.Errorf("Got %d, expected %d for GeneratePayload() as payload length", len(payload),
				test.expectedResLen)
		}
	}
}

// --- max logging for tests

func init() {
	log.SetLogLevel(log.Debug)
}
