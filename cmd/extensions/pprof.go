// Copyright 2018 Google LLC All Rights Reserved.
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

// +build profile

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
)

func init() {
	go func() {
		logrus.WithError(http.ListenAndServe(":6060", nil)).Info("Closed pprof server")
	}()
	logrus.Info("*** PPROF PROFILER STARTED on :6060 ***")
}
