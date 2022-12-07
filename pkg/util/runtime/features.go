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

	// FeatureCustomFasSyncInterval is a feature flag that enables a custom FleetAutoscaler resync interval
	FeatureCustomFasSyncInterval Feature = "CustomFasSyncInterval"

	// FeatureStateAllocationFilter is a feature flag that enables state filtering on Allocation.
	FeatureStateAllocationFilter Feature = "StateAllocationFilter"

	////////////////
	// Alpha features

	// FeatureLifecycleContract enables the `lifecycleContract` API to specify disruption tolerance.
	FeatureLifecycleContract Feature = "LifecycleContract"

	// FeaturePlayerAllocationFilter is a feature flag that enables the ability for Allocations to filter based on
	// player capacity.
	FeaturePlayerAllocationFilter Feature = "PlayerAllocationFilter"

	// FeaturePlayerTracking is a feature flag to enable/disable player tracking features.
	FeaturePlayerTracking Feature = "PlayerTracking"

	// FeatureResetMetricsOnDelete is a feature flag that tells the metrics service to unregister and register
	// relevant metric views to reset their state immediately when an Agones resource is deleted.
	FeatureResetMetricsOnDelete Feature = "ResetMetricsOnDelete"

	// FeatureSDKGracefulTermination is a feature flag that enables SDK to support gracefulTermination
	FeatureSDKGracefulTermination Feature = "SDKGracefulTermination"

	////////////////
	// Example feature

	// FeatureExample is an example feature gate flag, used for testing and demonstrative purposes
	FeatureExample Feature = "Example"
)

var (
	// featureDefaults is a map of all Feature Gates that are
	// operational in Agones, and what their default configuration is.
	// alpha features are disabled by default; beta features are enabled.
	//
	// To add a new alpha feature:
	// * add a const above
	// * add it to `featureDefaults`
	// * add it to install/helm/agones/defaultfeaturegates.yaml
	// * add it to `ALPHA_FEATURE_GATES` in build/Makefile
	// * add the inverse to the e2e-runner config in cloudbuild.yaml
	// * add it to site/content/en/docs/Guides/feature-stages.md
	//
	// To promote a feature from alpha->beta:
	// * move from `false` to `true` in `featureDefaults`
	// * move from `false` to `true` in install/helm/agones/defaultfeaturegates.yaml
	// * remove from `ALPHA_FEATURE_GATES` in build/Makefile
	// * invert in the e2e-runner config in cloudbuild.yaml
	// * change the value in site/content/en/docs/Guides/feature-stages.md
	//
	// To promote a feature from beta->GA:
	// * remove all places consuming the feature gate and fold logic to true
	//   * consider cleanup - often folding a gate to true allows refactoring
	// * invert the "new alpha feature" steps above
	//
	// In each of these, keep the feature sorted by descending maturity then alphabetical
	featureDefaults = map[Feature]bool{
		// Beta features
		FeatureCustomFasSyncInterval: true,
		FeatureStateAllocationFilter: true,

		// Alpha features
		FeatureLifecycleContract:      false,
		FeaturePlayerAllocationFilter: false,
		FeaturePlayerTracking:         false,
		FeatureResetMetricsOnDelete:   false,
		FeatureSDKGracefulTermination: false,

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
