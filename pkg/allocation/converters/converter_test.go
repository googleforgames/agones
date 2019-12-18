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

	pb "agones.dev/agones/pkg/allocation/go/v1alpha1"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/stretchr/testify/assert"
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
					Required: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"c": "d",
						},
					},
					Preferred: []metav1.LabelSelector{
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
				MetaPatch:                    &pb.MetaPatch{},
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
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := ConvertAllocationRequestV1Alpha1ToGSAV1(tc.in)
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}

			if !tc.skipConvertFromGSA {
				gsa := ConvertGSAV1ToAllocationRequestV1Alpha1(tc.want)
				if !assert.Equal(t, tc.in, gsa) {
					t.Errorf("mismatch with input after double conversion \"%s\"", tc.name)
				}
			}
		})
	}
}

func TestConvertGSAV1ToAllocationRequestV1Alpha1Empty(t *testing.T) {
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
				MetaPatch:                  &pb.MetaPatch{},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gsa := ConvertGSAV1ToAllocationRequestV1Alpha1(tc.in)
			if !assert.Equal(t, tc.want, gsa) {
				t.Errorf("mismatch with want after conversion \"%s\"", tc.name)
			}
		})
	}
}

func TestConvertGSAV1ToAllocationResponseV1Alpha1(t *testing.T) {
	tests := []struct {
		name             string
		in               *allocationv1.GameServerAllocation
		want             *pb.AllocationResponse
		skipConvertToGSA bool
	}{
		{
			name: "status field is set",
			in: &allocationv1.GameServerAllocation{
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
			want: &pb.AllocationResponse{
				State:          pb.AllocationResponse_UnAllocated,
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
			name: "status state is set to allocated",
			in: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationAllocated,
				},
			},
			want: &pb.AllocationResponse{
				State: pb.AllocationResponse_Allocated,
			},
		},
		{
			name: "status state is set to contention",
			in: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationContention,
				},
			},
			want: &pb.AllocationResponse{
				State: pb.AllocationResponse_Contention,
			},
		},
		{
			name: "Empty fields",
			in: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					Ports: []agonesv1.GameServerStatusPort{},
				},
			},
			want: &pb.AllocationResponse{
				State: pb.AllocationResponse_Unknown,
			},
			skipConvertToGSA: true,
		},
		{
			name: "Empty objects",
			in:   &allocationv1.GameServerAllocation{},
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

			out := ConvertGSAV1ToAllocationResponseV1Alpha1(tc.in)
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}

			if !tc.skipConvertToGSA {
				gsa := ConvertAllocationResponseV1Alpha1ToGSAV1(tc.want)
				if !assert.Equal(t, tc.in, gsa) {
					t.Errorf("mismatch with input after double conversion \"%s\"", tc.name)
				}
			}
		})
	}
}

func TestConvertAllocationResponseV1Alpha1ToGSAV1(t *testing.T) {
	tests := []struct {
		name string
		in   *pb.AllocationResponse
		want *allocationv1.GameServerAllocation
	}{
		{
			name: "Empty fields",
			in: &pb.AllocationResponse{
				State: pb.AllocationResponse_Unknown,
				Ports: []*pb.AllocationResponse_GameServerStatusPort{},
			},
			want: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := ConvertAllocationResponseV1Alpha1ToGSAV1(tc.in)
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}
		})
	}
}
