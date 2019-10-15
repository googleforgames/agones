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

// Package converters includes API conversions between GameServerAllocation API and the Allocation proto APIs.
package converters

import (
	pb "agones.dev/agones/pkg/allocation/go/v1alpha1"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertAllocationRequestV1Alpha1ToGSAV1 converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertAllocationRequestV1Alpha1ToGSAV1(in *pb.AllocationRequest) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.GetNamespace(),
		},
		Spec: allocationv1.GameServerAllocationSpec{
			Preferred:  convertLabelSelectorsFromReference(in.GetPreferredGameServerSelectors()),
			Scheduling: convertAllocationSchedulingV1Alpha1ToSchedulingStrategyV1(in.GetScheduling()),
		},
	}

	if in.GetMultiClusterSetting() != nil {
		gsa.Spec.MultiClusterSetting = allocationv1.MultiClusterSetting{
			Enabled: in.GetMultiClusterSetting().GetEnabled(),
		}
		if in.GetMultiClusterSetting().GetPolicySelector() != nil {
			gsa.Spec.MultiClusterSetting.PolicySelector = *in.GetMultiClusterSetting().GetPolicySelector()
		}
	}

	if in.GetMetaPatch() != nil {
		gsa.Spec.MetaPatch = allocationv1.MetaPatch{
			Labels:      in.GetMetaPatch().GetLabels(),
			Annotations: in.GetMetaPatch().GetAnnotations(),
		}
	}

	if in.GetRequiredGameServerSelector() != nil {
		gsa.Spec.Required = *in.RequiredGameServerSelector
	}
	return gsa
}

// ConvertGSAV1ToAllocationRequestV1Alpha1 converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertGSAV1ToAllocationRequestV1Alpha1(in *allocationv1.GameServerAllocation) *pb.AllocationRequest {
	if in == nil {
		return nil
	}

	out := &pb.AllocationRequest{
		Namespace:                    in.GetNamespace(),
		PreferredGameServerSelectors: convertLabelSelectorsToReference(in.Spec.Preferred),
		Scheduling:                   convertSchedulingStrategyV1ToAllocationSchedulingV1Alpha1(in.Spec.Scheduling),
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: in.Spec.MultiClusterSetting.Enabled,
		},
		RequiredGameServerSelector: &in.Spec.Required,
		MetaPatch: &pb.MetaPatch{
			Labels:      in.Spec.MetaPatch.Labels,
			Annotations: in.Spec.MetaPatch.Annotations,
		},
	}

	if in.Spec.MultiClusterSetting.Enabled {
		out.MultiClusterSetting.PolicySelector = &in.Spec.MultiClusterSetting.PolicySelector
	}

	return out
}

// convertAllocationSchedulingV1Alpha1ToSchedulingStrategyV1 converts AllocationRequest_SchedulingStrategy to apis.SchedulingStrategy
func convertAllocationSchedulingV1Alpha1ToSchedulingStrategyV1(in pb.AllocationRequest_SchedulingStrategy) apis.SchedulingStrategy {
	switch in {
	case pb.AllocationRequest_Packed:
		return apis.Packed
	case pb.AllocationRequest_Distributed:
		return apis.Distributed
	}
	return apis.Packed
}

// convertSchedulingStrategyV1ToAllocationSchedulingV1Alpha1 converts  apis.SchedulingStrategy to pb.AllocationRequest_SchedulingStrategy
func convertSchedulingStrategyV1ToAllocationSchedulingV1Alpha1(in apis.SchedulingStrategy) pb.AllocationRequest_SchedulingStrategy {
	switch in {
	case apis.Packed:
		return pb.AllocationRequest_Packed
	case apis.Distributed:
		return pb.AllocationRequest_Distributed
	}
	return pb.AllocationRequest_Packed
}

// convertLabelSelectorsFromReference converts []* to [] for LabelSelector
func convertLabelSelectorsFromReference(in []*metav1.LabelSelector) []metav1.LabelSelector {
	var result []metav1.LabelSelector
	for _, l := range in {
		result = append(result, *l)
	}
	return result
}

// convertLabelSelectorsToReference converts [] to []* for LabelSelector
func convertLabelSelectorsToReference(in []metav1.LabelSelector) []*metav1.LabelSelector {
	var result []*metav1.LabelSelector
	for _, l := range in {
		l := l
		result = append(result, &l)
	}
	return result
}

// ConvertGSAV1ToAllocationResponseV1Alpha1 converts GameServerAllocation V1 (GSA) to AllocationResponse
func ConvertGSAV1ToAllocationResponseV1Alpha1(in *allocationv1.GameServerAllocation) *pb.AllocationResponse {
	if in == nil {
		return nil
	}

	return &pb.AllocationResponse{
		State:          convertStateV1ToAllocationStateV1Alpha1(in.Status.State),
		GameServerName: in.Status.GameServerName,
		Address:        in.Status.Address,
		NodeName:       in.Status.NodeName,
		Ports:          convertAgonesPortsV1ToAllocationPortsV1Alpha1(in.Status.Ports),
	}
}

// ConvertAllocationResponseV1Alpha1ToGSAV1 converts AllocationResponse to GameServerAllocation V1 (GSA)
func ConvertAllocationResponseV1Alpha1ToGSAV1(in *pb.AllocationResponse) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	out := &allocationv1.GameServerAllocation{
		Status: allocationv1.GameServerAllocationStatus{
			GameServerName: in.GameServerName,
			Address:        in.Address,
			NodeName:       in.NodeName,
			Ports:          convertAllocationPortsV1Alpha1ToAgonesPortsV1(in.Ports),
		},
	}
	if state, isMatch := convertAllocationStateV1Alpha1ToStateV1(in.State); isMatch {
		out.Status.State = state
	}
	return out
}

// convertAgonesPortsV1ToAllocationPortsV1Alpha1 converts GameServerStatusPort V1 (GSA) to AllocationResponse_GameServerStatusPort
func convertAgonesPortsV1ToAllocationPortsV1Alpha1(in []agonesv1.GameServerStatusPort) []*pb.AllocationResponse_GameServerStatusPort {
	var pbPorts []*pb.AllocationResponse_GameServerStatusPort
	for _, port := range in {
		pbPort := &pb.AllocationResponse_GameServerStatusPort{
			Name: port.Name,
			Port: port.Port,
		}
		pbPorts = append(pbPorts, pbPort)
	}
	return pbPorts
}

// convertAllocationPortsV1Alpha1ToAgonesPortsV1 converts AllocationResponse_GameServerStatusPort to GameServerStatusPort V1 (GSA)
func convertAllocationPortsV1Alpha1ToAgonesPortsV1(in []*pb.AllocationResponse_GameServerStatusPort) []agonesv1.GameServerStatusPort {
	var out []agonesv1.GameServerStatusPort
	for _, port := range in {
		p := &agonesv1.GameServerStatusPort{
			Name: port.Name,
			Port: port.Port,
		}
		out = append(out, *p)
	}
	return out
}

// convertStateV1ToAllocationStateV1Alpha1 converts GameServerAllocationState V1 (GSA) to AllocationResponse_GameServerAllocationState
func convertStateV1ToAllocationStateV1Alpha1(in allocationv1.GameServerAllocationState) pb.AllocationResponse_GameServerAllocationState {
	switch in {
	case allocationv1.GameServerAllocationAllocated:
		return pb.AllocationResponse_Allocated
	case allocationv1.GameServerAllocationUnAllocated:
		return pb.AllocationResponse_UnAllocated
	case allocationv1.GameServerAllocationContention:
		return pb.AllocationResponse_Contention
	}
	return pb.AllocationResponse_Unknown
}

// convertAllocationStateV1Alpha1ToStateV1 converts AllocationResponse_GameServerAllocationState to GameServerAllocationState V1 (GSA)
// if the conversion cannot happen, it returns false.
func convertAllocationStateV1Alpha1ToStateV1(in pb.AllocationResponse_GameServerAllocationState) (allocationv1.GameServerAllocationState, bool) {
	switch in {
	case pb.AllocationResponse_Allocated:
		return allocationv1.GameServerAllocationAllocated, true
	case pb.AllocationResponse_UnAllocated:
		return allocationv1.GameServerAllocationUnAllocated, true
	case pb.AllocationResponse_Contention:
		return allocationv1.GameServerAllocationContention, true
	}
	return allocationv1.GameServerAllocationUnAllocated, false
}
