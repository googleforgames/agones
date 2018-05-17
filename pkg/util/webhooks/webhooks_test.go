// Copyright 2018 Google Inc. All Rights Reserved.
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

package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"encoding/json"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testServer struct {
	server *httptest.Server
}

func (ts *testServer) Close() error {
	ts.server.Close()
	return nil
}

// ListenAndServeTLS(certFile, keyFile string) error
func (ts *testServer) ListenAndServeTLS(certFile, keyFile string) error {
	ts.server.StartTLS()
	return nil
}

func TestWebHookAddHandler(t *testing.T) {
	t.Parallel()

	type testHandler struct {
		gk schema.GroupKind
		op v1beta1.Operation
	}
	type expected struct {
		count int
	}
	fixtures := map[string]struct {
		handlers []testHandler
		expected expected
	}{
		"single, matching": {
			handlers: []testHandler{{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create}},
			expected: expected{count: 1},
		},
		"double, one matching op": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Update},
			},
		},
		"double, one matching group": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create},
				{gk: schema.GroupKind{Group: "nope", Kind: "kind"}, op: v1beta1.Create},
			},
		},
		"double, one matching kind": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "nope"}, op: v1beta1.Create},
			},
		},
		"double, both matching": {
			expected: expected{count: 2},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: v1beta1.Create},
			},
		},
	}

	for k, handles := range fixtures {
		t.Run(k, func(t *testing.T) {

			stop := make(chan struct{})
			defer close(stop)

			fixture := v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{
				Kind:      v1.GroupVersionKind{Kind: "kind", Group: "group", Version: "version"},
				Operation: v1beta1.Create,
				UID:       "1234"}}

			callCount := 0
			wh := NewWebHook("", "")
			ts := &testServer{server: httptest.NewUnstartedServer(wh.mux)}
			wh.server = ts

			for _, th := range handles.handlers {
				wh.AddHandler("/test", th.gk, th.op, func(review v1beta1.AdmissionReview) (v1beta1.AdmissionReview, error) {
					assert.Equal(t, fixture.Request, review.Request)
					assert.True(t, review.Response.Allowed)
					callCount++
					return review, nil
				})
			}

			err := wh.Run(0, stop)
			assert.Nil(t, err)

			client := ts.server.Client()
			url := ts.server.URL + "/test"

			buf := &bytes.Buffer{}
			err = json.NewEncoder(buf).Encode(fixture)
			assert.Nil(t, err)

			r, err := http.NewRequest("GET", url, buf)
			assert.Nil(t, err)

			resp, err := client.Do(r)
			assert.Nil(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, handles.expected.count, callCount, "[%v] /test should have been called for %#v", k, handles)
		})
	}

}
