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

package e2e

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// defaultGSS returns a default GameServerSet configuration
func defaultGSS() *agonesv1.GameServerSet {
	gs := defaultGameServer()
	return gssWithGameServerSpec(gs.Spec)
}

// gssWithGameServerSpec returns a GameServerSet with specified gameserver spec
func gssWithGameServerSpec(gsSpec agonesv1.GameServerSpec) *agonesv1.GameServerSet {
	return &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Name: "GameServerSet", Namespace: defaultNs},
		Spec: agonesv1.GameServerSetSpec{
			Replicas: replicasCount,
			Template: agonesv1.GameServerTemplateSpec{
				Spec: gsSpec,
			},
		},
	}
}

// TestGSSNameValidation is built to test GameServerSet Name length validation,
// GameServerSet Name should have at most 63 chars.
// nolint:dupl
func TestGSSNameValidation(t *testing.T) {
	t.Parallel()
	client := framework.AgonesClient.AgonesV1()

	gss := defaultGSS()
	nameLen := validation.LabelValueMaxLength + 1
	bytes := make([]byte, nameLen)
	for i := 0; i < nameLen; i++ {
		bytes[i] = 'f'
	}
	gss.Name = string(bytes)
	_, err := client.GameServerSets(defaultNs).Create(gss)
	assert.NotNil(t, err)
	statusErr, ok := err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.True(t, len(statusErr.Status().Details.Causes) > 0)
	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, statusErr.Status().Details.Causes[0].Type)
	goodGss := defaultGSS()
	goodGss.Name = string(bytes[0 : nameLen-1])
	goodGss, err = client.GameServerSets(defaultNs).Create(goodGss)
	if assert.Nil(t, err) {
		defer client.GameServerSets(defaultNs).Delete(goodGss.ObjectMeta.Name, nil) // nolint:errcheck
	}
}
