// Copyright 2017 Google LLC All Rights Reserved.
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

// Package signals contains utilities for managing process signals,
// particularly around stopping processes
package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// NewSigKillContext returns a Context that cancels when os.Interrupt or os.Kill is received
func NewSigKillContext() (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	return ctx, cancel
}

// NewSigTermHandler creates a channel to listen to SIGTERM and runs the handle function
func NewSigTermHandler(handle func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		handle()
	}()
}
