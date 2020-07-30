// Copyright 2020 Google LLC All Rights Reserved.
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

package fuzz

// This file holds fuzzers that are implemented
// using go-fuzz. More info about go-fuzz can
// be found at https://github.com/dvyukov/go-fuzz

// To run the below fuzzer locally, follow these steps:
// 1) go get -u github.com/dvyukov/go-fuzz/go-fuzz
// 2) go get -u github.com/dvyukov/go-fuzz/go-fuzz-build
// 3) cd to directory of fuzz.go
// 4) $GOPATH/bin/go-fuzz-build
// 5) $GOPATH/bin/go-fuzz

import "agones.dev/agones/pkg/util/runtime"

// Fuzz implements a fuzzer that targets ParseFeatures
func Fuzz(data []byte) int {
	err := runtime.ParseFeatures(string(data))
	if err != nil {
		return 0
	}
	return 1
}
