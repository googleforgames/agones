// Copyright 2019 Google LLC All Rights Reserved.
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
	"fmt"
	"strings"

	"agones.dev/agones/pkg/util/runtime"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyName       = MustTagKey("name")
	keyFleetName  = MustTagKey("fleet_name")
	keyType       = MustTagKey("type")
	keyStatusCode = MustTagKey("status_code")
	keyVerb       = MustTagKey("verb")
	keyEndpoint   = MustTagKey("endpoint")
	keyEmpty      = MustTagKey("empty")
)

func recordWithTags(ctx context.Context, mutators []tag.Mutator, ms ...stats.Measurement) {
	if err := stats.RecordWithTags(ctx, mutators, ms...); err != nil {
		logger.WithError(err).Warn("error while recoding stats")
	}
}

// MustTagKey creates a new `tag.Key` from a string, panic if the key is not a valid.
func MustTagKey(key string) tag.Key {
	t, err := tag.NewKey(key)
	if err != nil {
		panic(err)
	}
	return t
}

func parseLabels(s string) (*stackdriver.Labels, error) {
	res := &stackdriver.Labels{}
	if s == "" {
		return res, nil
	}
	pairs := strings.Split(s, ",")
	if len(pairs) == 0 {
		return res, nil
	}
	for _, p := range pairs {
		keyValue := strings.Split(p, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid labels format: %s, expect key=value,key2=value2", s)
		}
		res.Set(keyValue[0], keyValue[1], "")
	}
	return res, nil
}
