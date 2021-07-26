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

package converters

import (
	"testing"

	pb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertAllocationRequestToGameServerAllocation(t *testing.T) {
	tests := []struct {
		name               string
		in                 *pb.AllocationRequest
		want               *allocationv1.GameServerAllocation
		skipConvertFromGSA bool
	}{
		{
			name: "all fields are set",
			in: &pb.AllocationRequest{
				Namespace: "ns",
				MultiClusterSetting: &pb.MultiClusterSetting{
					Enabled: true,
					PolicySelector: &pb.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
				},
				RequiredGameServerSelector: &pb.LabelSelector{
					MatchLabels: map[string]string{
						"c": "d",
					},
				},
				PreferredGameServerSelectors: []*pb.LabelSelector{
					{
						MatchLabels: map[string]string{
							"e": "f",
						},
					},
					{
						MatchLabels: map[string]string{
							"g": "h",
						},
					},
				},
				Scheduling: pb.AllocationRequest_Packed,
				Metadata: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: true,
						PolicySelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a": "b",
							},
						},
					},
					Required: allocationv1.GameServerSelector{
						LabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"c": "d",
							},
						}},
					Preferred: []allocationv1.GameServerSelector{
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"e": "f",
								},
							}},
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"g": "h",
								},
							}},
					},
					Scheduling: apis.Packed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"i": "j",
						},
					},
				},
			},
		},
		{
			name: "empty fields to GSA",
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.LabelSelector{},
				PreferredGameServerSelectors: []*pb.LabelSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				Metadata:                     &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
				},
			},
			skipConvertFromGSA: true,
		},
		{
			name: "empty object to GSA",
			in:   &pb.AllocationRequest{},
			want: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: apis.Packed,
				},
			},
			skipConvertFromGSA: true,
		},
		{
			name: "nil object",
			in:   nil,
			want: nil,
		},
		{
			name: "accepts deprecated metapatch field",
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.LabelSelector{},
				PreferredGameServerSelectors: []*pb.LabelSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"a": "b",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"a": "b",
						},
					},
				},
			},
			skipConvertFromGSA: true,
		},
		{
			name: "Prefers metadata over metapatch field",
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.LabelSelector{},
				PreferredGameServerSelectors: []*pb.LabelSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				Metadata: &pb.MetaPatch{
					Labels: map[string]string{
						"a": "b",
					},
				},
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"c": "d",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"a": "b",
						},
					},
				},
			},
			skipConvertFromGSA: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := ConvertAllocationRequestToGSA(tc.in)
			assert.Equal(t, tc.want, out, "mismatch with want after conversion: \"%s\"", tc.name)

			if !tc.skipConvertFromGSA {
				gsa := ConvertGSAToAllocationRequest(tc.want)
				assert.Equal(t, tc.in, gsa, "mismatch with input after double conversion \"%s\"", tc.name)
			}
		})
	}
}

func TestConvertGSAToAllocationRequestEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   *allocationv1.GameServerAllocation
		want *pb.AllocationRequest
	}{
		{
			name: "empty fields",
			in: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
				},
			},
			want: &pb.AllocationRequest{
				Namespace:                  "",
				MultiClusterSetting:        &pb.MultiClusterSetting{},
				RequiredGameServerSelector: &pb.LabelSelector{},
				Scheduling:                 pb.AllocationRequest_Distributed,
				Metadata:                   &pb.MetaPatch{},
				MetaPatch:                  &pb.MetaPatch{},
			},
		}, {
			name: "empty object",
			in: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: apis.Packed,
				},
			},
			want: &pb.AllocationRequest{
				MultiClusterSetting:        &pb.MultiClusterSetting{},
				RequiredGameServerSelector: &pb.LabelSelector{},
				Metadata:                   &pb.MetaPatch{},
				MetaPatch:                  &pb.MetaPatch{},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gsa := ConvertGSAToAllocationRequest(tc.in)
			if !assert.Equal(t, tc.want, gsa) {
				t.Errorf("mismatch with want after conversion \"%s\"", tc.name)
			}
		})
	}
}

func TestConvertGSAToAllocationResponse(t *testing.T) {
	tests := []struct {
		name             string
		in               *allocationv1.GameServerAllocation
		want             *pb.AllocationResponse
		wantErrCode      codes.Code
		skipConvertToGSA bool
	}{
		{
			name: "status state is set to allocated",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				NodeName:       "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
			},
		},
		{
			name: "status field is set to unallocated",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationUnAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
				},
			},
			wantErrCode:      codes.ResourceExhausted,
			skipConvertToGSA: true,
		},
		{
			name: "status state is set to contention",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationContention,
				},
			},
			wantErrCode:      codes.Aborted,
			skipConvertToGSA: true,
		},
		{
			name: "Empty fields",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					Ports: []agonesv1.GameServerStatusPort{},
				},
			},
			wantErrCode:      codes.Unknown,
			skipConvertToGSA: true,
		},
		{
			name: "Empty objects",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationAllocated,
				},
			},
			want: &pb.AllocationResponse{},
		},
		{
			name: "nil objects",
			in:   nil,
			want: nil,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := ConvertGSAToAllocationResponse(tc.in)
			if tc.wantErrCode != 0 {
				st, ok := status.FromError(err)
				if !assert.True(t, ok) {
					return
				}
				assert.Equal(t, tc.wantErrCode, st.Code())
			}
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}

			if !tc.skipConvertToGSA {
				gsa := ConvertAllocationResponseToGSA(tc.want)
				if !assert.Equal(t, tc.in, gsa) {
					t.Errorf("mismatch with input after double conversion \"%s\"", tc.name)
				}
			}
		})
	}
}

func TestConvertAllocationResponseToGSA(t *testing.T) {
	tests := []struct {
		name string
		in   *pb.AllocationResponse
		want *allocationv1.GameServerAllocation
	}{
		{
			name: "Empty fields",
			in: &pb.AllocationResponse{
				Ports: []*pb.AllocationResponse_GameServerStatusPort{},
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationAllocated,
				},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := ConvertAllocationResponseToGSA(tc.in)
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}
		})
	}
}
