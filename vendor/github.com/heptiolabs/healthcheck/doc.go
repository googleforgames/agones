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

/*
Package healthcheck helps you implement Kubernetes liveness and readiness checks
for your application. It supports synchronous and asynchronous (background)
checks. It can optionally report each check's status as a set of Prometheus
gauge metrics for cluster-wide monitoring and alerting.

It also includes a small library of generic checks for DNS, TCP, and HTTP
reachability as well as Goroutine usage.
*/
package healthcheck
