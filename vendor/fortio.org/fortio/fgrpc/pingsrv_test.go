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

package fgrpc

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"

	"fortio.org/fortio/fnet"
	"fortio.org/fortio/log"
)

func init() {
	log.SetLogLevel(log.Debug)
}

func TestPingServer(t *testing.T) {
	iPort := PingServerTCP("0", "", "", "foo", 0)
	iAddr := fmt.Sprintf("localhost:%d", iPort)
	t.Logf("insecure grpc ping server running, will connect to %s", iAddr)
	sPort := PingServerTCP("0", svrCrt, svrKey, "foo", 0)
	sAddr := fmt.Sprintf("localhost:%d", sPort)
	t.Logf("secure grpc ping server running, will connect to %s", sAddr)
	delay := 100 * time.Millisecond
	latency, err := PingClientCall(iAddr, "", 7, "test payload", delay)
	if err != nil || latency < delay.Seconds() || latency > 10.*delay.Seconds() {
		t.Errorf("Unexpected result %f, %v with ping calls and delay of %v", latency, err, delay)
	}
	if latency, err := PingClientCall(fnet.PrefixHTTPS+"fortio.istio.io:443", "", 7,
		"test payload", 0); err != nil || latency <= 0 {
		t.Errorf("Unexpected result %f, %v with ping calls", latency, err)
	}
	if latency, err := PingClientCall(sAddr, caCrt, 7, "test payload", 0); err != nil || latency <= 0 {
		t.Errorf("Unexpected result %f, %v with ping calls", latency, err)
	}
	if latency, err := PingClientCall(iAddr, caCrt, 1, "", 0); err == nil {
		t.Errorf("Should have had an error instead of result %f for secure ping to insecure port", latency)
	}
	if latency, err := PingClientCall(sAddr, "", 1, "", 0); err == nil {
		t.Errorf("Should have had an error instead of result %f for insecure ping to secure port", latency)
	}
	if creds, err := credentials.NewServerTLSFromFile(failCrt, failKey); err == nil {
		t.Errorf("Should have had an error instead of result %f for ping server", creds)
	}
	serving := grpc_health_v1.HealthCheckResponse_SERVING.String()
	if r, err := GrpcHealthCheck(iAddr, "", "", 1); err != nil || (*r)[serving] != 1 {
		t.Errorf("Unexpected result %+v, %v with empty service health check", r, err)
	}
	if r, err := GrpcHealthCheck(sAddr, caCrt, "", 1); err != nil || (*r)[serving] != 1 {
		t.Errorf("Unexpected result %+v, %v with empty service health check", r, err)
	}
	if r, err := GrpcHealthCheck(fnet.PrefixHTTPS+"fortio.istio.io:443", "", "", 1); err != nil || (*r)[serving] != 1 {
		t.Errorf("Unexpected result %+v, %v with empty service health check", r, err)
	}
	if r, err := GrpcHealthCheck(iAddr, "", "foo", 3); err != nil || (*r)[serving] != 3 {
		t.Errorf("Unexpected result %+v, %v with health check for same service as started (foo)", r, err)
	}
	if r, err := GrpcHealthCheck(sAddr, caCrt, "foo", 3); err != nil || (*r)[serving] != 3 {
		t.Errorf("Unexpected result %+v, %v with health check for same service as started (foo)", r, err)
	}
	if r, err := GrpcHealthCheck(iAddr, "", "willfail", 1); err == nil || r != nil {
		t.Errorf("Was expecting error when using unknown service, didn't get one, got %+v", r)
	}
	if r, err := GrpcHealthCheck(sAddr, caCrt, "willfail", 1); err == nil || r != nil {
		t.Errorf("Was expecting error when using unknown service, didn't get one, got %+v", r)
	}
	if r, err := GrpcHealthCheck(sAddr, failCrt, "willfail", 1); err == nil {
		t.Errorf("Was expecting dial error when using invalid certificate, didn't get one, got %+v", r)
	}
	// 2nd server on same port should fail to bind:
	newPort := PingServerTCP(strconv.Itoa(iPort), "", "", "will fail", 0)
	if newPort != -1 {
		t.Errorf("Didn't expect 2nd server on same port to succeed: %d %d", newPort, iPort)
	}
}
