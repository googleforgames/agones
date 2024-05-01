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
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// FeatureGateFlag is a name of a command line flag, which turns on specific tests for FeatureGates
	FeatureGateFlag = "feature-gates"

	////////////////
	// Beta features

	// FeatureCountsAndLists is a feature flag that enables counts and lists feature
	// (a generic implenetation of the player tracking feature).
	FeatureCountsAndLists Feature = "CountsAndLists"

	////////////////
	// Alpha features

	// FeatureDisableResyncOnSDKServer is a feature flag to enable/disable resync on SDK server.
	FeatureDisableResyncOnSDKServer Feature = "DisableResyncOnSDKServer"

	////////////////
	// Alpha features

	// FeatureGKEAutopilotExtendedDurationPods enables the use of Extended Duration pods
	// when Agones is running on Autopilot. Available on 1.28+ only.
	FeatureGKEAutopilotExtendedDurationPods = "GKEAutopilotExtendedDurationPods"

	// FeaturePlayerAllocationFilter is a feature flag that enables the ability for Allocations to filter based on
	// player capacity.
	FeaturePlayerAllocationFilter Feature = "PlayerAllocationFilter"

	// FeaturePlayerTracking is a feature flag to enable/disable player tracking features.
	FeaturePlayerTracking Feature = "PlayerTracking"

	////////////////
	// Dev features

	// FeatureAutopilotPassthroughPort is a feature flag that enables/disables Passthrough Port Policy.
	FeatureAutopilotPassthroughPort Feature = "PassthroughPortPolicy"

	////////////////
	// Example feature

	// FeatureExample is an example feature gate flag, used for testing and demonstrative purposes
	FeatureExample Feature = "Example"
)

var (
	// featureDefaults is a map of all Feature Gates that are
	// operational in Agones, and what their default configuration is.
	// dev & alpha features are disabled by default; beta features are enabled.
	//
	// To add a new dev feature (an in progress feature, not tested in CI and not publicly documented):
	// * add a const above
	// * add it to `featureDefaults`
	// * add it to install/helm/agones/defaultfeaturegates.yaml
	// * note: you can add a new feature as an alpha feature if you're ready to test it in CI
	//
	// To promote a feature from dev->alpha:
	// * add it to `ALPHA_FEATURE_GATES` in build/Makefile
	// * add the inverse to the e2e-runner config in cloudbuild.yaml
	// * add it to site/content/en/docs/Guides/feature-stages.md
	// * Ensure that the features in each file are organized categorically and alphabetically.
	//
	// To promote a feature from alpha->beta:
	// * move from `false` to `true` in `featureDefaults`.
	// * move from `false` to `true` in install/helm/agones/defaultfeaturegates.yaml
	// * remove from `ALPHA_FEATURE_GATES` in build/Makefile
	// * add to `BETA_FEATURE_GATES` in build/Makefile
	// * invert in the e2e-runner config in cloudbuild.yaml
	// * change the value in site/content/en/docs/Guides/feature-stages.md.
	// * Ensure that the features in each file are organized categorically and alphabetically.
	//
	// To promote a feature from beta->GA:
	// * remove all places consuming the feature gate and fold logic to true
	// * consider cleanup - often folding a gate to true allows refactoring
	// * invert the "new alpha feature" steps above
	// * remove from `BETA_FEATURE_GATES` in build/Makefile
	//
	// In each of these, keep the feature sorted by descending maturity then alphabetical
	featureDefaults = map[Feature]bool{
		// Beta features
		FeatureCountsAndLists:           true,
		FeatureDisableResyncOnSDKServer: true,

		// Alpha features
		FeatureGKEAutopilotExtendedDurationPods: false,
		FeaturePlayerAllocationFilter:           false,
		FeaturePlayerTracking:                   false,

		// Dev features
		FeatureAutopilotPassthroughPort: true,

		// Example feature
		FeatureExample: false,
	}

	// featureGates is the storage of what features are enabled
	// or disabled.
	featureGates map[Feature]bool

	// featureMutex ensures that updates to featureGates don't happen at the same time as reads.
	// this is mostly to protect tests which can change gates in parallel.
	featureMutex = sync.RWMutex{}

	// FeatureTestMutex is a mutex to be shared between tests to ensure that a test that involves changing featureGates
	// cannot accidentally run at the same time as another test that also changing feature flags.
	FeatureTestMutex sync.Mutex
)

// Feature is a type for defining feature gates.
type Feature string

// FeaturesBindFlags does the Viper arguments configuration. Call before running pflag.Parse()
func FeaturesBindFlags() {
	viper.SetDefault(FeatureGateFlag, "")
	pflag.String(FeatureGateFlag, viper.GetString(FeatureGateFlag), "Flag to pass in the url query list of feature flags to enable or disable")
}

// FeaturesBindEnv binds the environment variables, based on the flags provided.
// call after viper.SetEnvKeyReplacer(...) if it is being set.
func FeaturesBindEnv() error {
	return viper.BindEnv(FeatureGateFlag)
}

// ParseFeaturesFromEnv will parse the feature flags from the Viper args
// configured by FeaturesBindFlags() and FeaturesBindEnv()
func ParseFeaturesFromEnv() error {
	return ParseFeatures(viper.GetString(FeatureGateFlag))
}

// ParseFeatures parses the url encoded query string of features and stores the value
// for later retrieval
func ParseFeatures(queryString string) error {
	featureMutex.Lock()
	defer featureMutex.Unlock()

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

// EnableAllFeatures turns on all feature flags.
// This is useful for libraries/processes/tests that want to
// enable all Alpha/Beta features without having to track all
// the current feature flags.
func EnableAllFeatures() {
	featureMutex.Lock()
	defer featureMutex.Unlock()

	features := map[Feature]bool{}
	// copy the defaults into this map
	for k := range featureDefaults {
		features[k] = true
	}

	featureGates = features
}

// FeatureEnabled returns if a Feature is enabled or not
func FeatureEnabled(feature Feature) bool {
	featureMutex.RLock()
	defer featureMutex.RUnlock()
	return featureGates[feature]
}

// EncodeFeatures returns the feature set as a URL encoded query string
func EncodeFeatures() string {
	values := url.Values{}
	featureMutex.RLock()
	defer featureMutex.RUnlock()

	for k, v := range featureGates {
		values.Add(string(k), strconv.FormatBool(v))
	}
	return values.Encode()
}
