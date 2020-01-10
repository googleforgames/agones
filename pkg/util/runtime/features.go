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

package runtime

import (
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	featureGateFlag = "feature-gates"

	// FeatureExample is an example feature gate flag, used for testing and demonstrative purposes
	FeatureExample Feature = "Example"
)

var (
	// featureDefaults is a map of all Feature Gates that are
	// operational in Agones, and what their default configuration is.
	featureDefaults = map[Feature]bool{
		FeatureExample: true,
	}

	// featureGates is the storage of what features are enabled
	// or disabled.
	featureGates map[Feature]bool
)

// Feature is a type for defining feature gates.
type Feature string

// FeaturesBindFlags does the Viper arguments configuration. Call before running pflag.Parse()
func FeaturesBindFlags() {
	viper.SetDefault(featureGateFlag, "")
	pflag.String(featureGateFlag, viper.GetString(featureGateFlag), "Flag to pass in the url query list of feature flags to enable or disable")
}

// FeaturesBindEnv binds the environment variables, based on the flags provided.
// call after viper.SetEnvKeyReplacer(...) if it is being set.
func FeaturesBindEnv() error {
	return viper.BindEnv(featureGateFlag)
}

// ParseFeaturesFromEnv will parse the feature flags from the Viper args
// configured by FeaturesBindFlags() and FeaturesBindEnv()
func ParseFeaturesFromEnv() error {
	return ParseFeatures(viper.GetString(featureGateFlag))
}

// ParseFeatures parses the url encoded query string of features and stores the value
// for later retrieval
func ParseFeatures(queryString string) error {
	features := map[Feature]bool{}
	// copy the defaults into this map
	for k, v := range featureDefaults {
		features[k] = v
	}

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return errors.Wrap(err, "error parsing query string for feature gates")
	}

	for k := range values {
		f := Feature(k)

		if _, ok := featureDefaults[f]; !ok {
			return errors.Errorf("Feature Gate %q is not a valid Feature Gate", f)
		}

		b, err := strconv.ParseBool(values.Get(k))
		if err != nil {
			return errors.Wrapf(err, "error parsing bool value from flag %s ", k)
		}
		features[f] = b
	}

	featureGates = features
	return nil
}

// FeatureEnabled returns if a Feature is enabled or not
func FeatureEnabled(feature Feature) bool {
	return featureGates[feature]
}

// EncodeFeatures returns the feature set as a URL encoded query string
func EncodeFeatures() string {
	values := url.Values{}
	for k, v := range featureGates {
		values.Add(string(k), strconv.FormatBool(v))
	}
	return values.Encode()
}
