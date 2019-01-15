// Copyright 2019 Google Inc. All Rights Reserved.
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

package metrics

import (
	"context"

	"agones.dev/agones/pkg/util/runtime"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyName       = mustTagKey("name")
	keyFleetName  = mustTagKey("fleet_name")
	keyType       = mustTagKey("type")
	keyStatusCode = mustTagKey("status_code")
	keyVerb       = mustTagKey("verb")
	keyEndpoint   = mustTagKey("endpoint")
)

func recordWithTags(ctx context.Context, mutators []tag.Mutator, ms ...stats.Measurement) {
	if err := stats.RecordWithTags(ctx, mutators, ms...); err != nil {
		logger.WithError(err).Warn("error while recoding stats")
	}
}

func mustTagKey(key string) tag.Key {
	t, err := tag.NewKey(key)
	if err != nil {
		panic(err)
	}
	return t
}
