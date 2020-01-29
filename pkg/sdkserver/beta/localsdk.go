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

package beta

import "agones.dev/agones/pkg/sdk/beta"

var _ beta.SDKServer = LocalSDKServer{}

// LocalSDKServer is the local sdk server implementation
// for beta features
type LocalSDKServer struct{}

// NewLocalSDKServer is a constructor for the beta local SDK Server
func NewLocalSDKServer() *LocalSDKServer {
	return &LocalSDKServer{}
}

// Close tears down all the things
func (l *LocalSDKServer) Close() {
	// placeholder in case things need to be shut down
}
