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
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
)

const sourceKey = "source"

// stackTracer is the pkg/errors stacktrace interface
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// replace the standard glog error logger, with a logrus one
func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	runtime.ErrorHandlers[0] = func(err error) {
		if stackTrace, ok := err.(stackTracer); ok {
			var stack []string
			for _, f := range stackTrace.StackTrace() {
				stack = append(stack, fmt.Sprintf("%+v", f))
			}
			logrus.WithField("stack", stack).Error(err)
		} else {
			logrus.Error(err)
		}
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

// NewLoggerWithSource returns a logrus.Entry to use when you want to specify an source
func NewLoggerWithSource(source string) *logrus.Entry {
	return logrus.WithField(sourceKey, source)
}

// NewLoggerWithType returns a logrus.Entry to use when you want to use a data type as the source
// such as when you have a struct with methods
func NewLoggerWithType(obj interface{}) *logrus.Entry {
	return NewLoggerWithSource(fmt.Sprintf("%T", obj))
}
