// Copyright 2017 Google Inc. All Rights Reserved.
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

// Package runtime handles runtime errors
// Wraps and reconfigures functionality in apimachinery/pkg/runtime
package runtime

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// replace the standard glog error logger, with a logrus one
func init() {
	runtime.ErrorHandlers[0] = func(err error) {
		logrus.Errorf("stack: %+v", err)
	}
}

// HandleError wraps runtime.HandleError so that it is possible to
// use WithField with logrus.
func HandleError(logger *logrus.Entry, err error) {
	// it's a bit of a double handle, but I can't see a better way to do it
	logger.WithError(err).Error()
	runtime.HandleError(err)
}

// Must panics if there is an error
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
