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
	pb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertAllocationRequestToGSA converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertAllocationRequestToGSA(in *pb.AllocationRequest) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.GetNamespace(),
		},
		Spec: allocationv1.GameServerAllocationSpec{
			Preferred:  convertLabelSelectorsToInternalLabelSelectors(in.GetPreferredGameServerSelectors()),
			Scheduling: convertAllocationSchedulingToGSASchedulingStrategy(in.GetScheduling()),
		},
	}

	if in.GetMultiClusterSetting() != nil {
		gsa.Spec.MultiClusterSetting = allocationv1.MultiClusterSetting{
			Enabled: in.GetMultiClusterSetting().GetEnabled(),
		}
		if ls := convertLabelSelectorToInternalLabelSelector(in.GetMultiClusterSetting().GetPolicySelector()); ls != nil {
			gsa.Spec.MultiClusterSetting.PolicySelector = *ls
		}
	}

	// Accept both metadata (preferred) and metapatch until metapatch is fully removed.
	metadata := in.GetMetadata()
	if metadata == nil {
		metadata = in.GetMetaPatch()
	}

	if metadata != nil {
		gsa.Spec.MetaPatch = allocationv1.MetaPatch{
			Labels:      metadata.GetLabels(),
			Annotations: metadata.GetAnnotations(),
		}
	}

	if ls := convertLabelSelectorToInternalLabelSelector(in.GetRequiredGameServerSelector()); ls != nil {
		gsa.Spec.Required = allocationv1.GameServerSelector{LabelSelector: *ls}
	}
	return gsa
}

// ConvertGSAToAllocationRequest converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertGSAToAllocationRequest(in *allocationv1.GameServerAllocation) *pb.AllocationRequest {
	if in == nil {
		return nil
	}

	out := &pb.AllocationRequest{
		Namespace:                    in.GetNamespace(),
		PreferredGameServerSelectors: convertInternalLabelSelectorsToLabelSelectors(in.Spec.Preferred),
		Scheduling:                   convertGSASchedulingStrategyToAllocationScheduling(in.Spec.Scheduling),
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: in.Spec.MultiClusterSetting.Enabled,
		},
		RequiredGameServerSelector: convertInternalLabelSelectorToLabelSelector(&in.Spec.Required.LabelSelector),
		Metadata: &pb.MetaPatch{
			Labels:      in.Spec.MetaPatch.Labels,
			Annotations: in.Spec.MetaPatch.Annotations,
		},
		// MetaPatch is deprecated, but we do a double write here to both metapatch and metadata
		// to ensure that multi-cluster allocation still works when one cluster has the field
		// and another one does not have the field yet.
		MetaPatch: &pb.MetaPatch{
			Labels:      in.Spec.MetaPatch.Labels,
			Annotations: in.Spec.MetaPatch.Annotations,
		},
	}

	if in.Spec.MultiClusterSetting.Enabled {
		out.MultiClusterSetting.PolicySelector = convertInternalLabelSelectorToLabelSelector(&in.Spec.MultiClusterSetting.PolicySelector)
	}

	return out
}

// convertAllocationSchedulingToGSASchedulingStrategy converts AllocationRequest_SchedulingStrategy to apis.SchedulingStrategy
func convertAllocationSchedulingToGSASchedulingStrategy(in pb.AllocationRequest_SchedulingStrategy) apis.SchedulingStrategy {
	switch in {
	case pb.AllocationRequest_Packed:
		return apis.Packed
	case pb.AllocationRequest_Distributed:
		return apis.Distributed
	}
	return apis.Packed
}

// convertGSASchedulingStrategyToAllocationScheduling converts  apis.SchedulingStrategy to pb.AllocationRequest_SchedulingStrategy
func convertGSASchedulingStrategyToAllocationScheduling(in apis.SchedulingStrategy) pb.AllocationRequest_SchedulingStrategy {
	switch in {
	case apis.Packed:
		return pb.AllocationRequest_Packed
	case apis.Distributed:
		return pb.AllocationRequest_Distributed
	}
	return pb.AllocationRequest_Packed
}

func convertLabelSelectorToInternalLabelSelector(in *pb.LabelSelector) *metav1.LabelSelector {
	if in == nil {
		return nil
	}
	return &metav1.LabelSelector{MatchLabels: in.GetMatchLabels()}
}

func convertInternalLabelSelectorToLabelSelector(in *metav1.LabelSelector) *pb.LabelSelector {
	if in == nil {
		return nil
	}
	return &pb.LabelSelector{MatchLabels: in.MatchLabels}
}

func convertInternalLabelSelectorsToLabelSelectors(in []allocationv1.GameServerSelector) []*pb.LabelSelector {
	var result []*pb.LabelSelector
	for _, l := range in {
		l := l
		c := convertInternalLabelSelectorToLabelSelector(&l.LabelSelector)
		result = append(result, c)
	}
	return result
}

func convertLabelSelectorsToInternalLabelSelectors(in []*pb.LabelSelector) []allocationv1.GameServerSelector {
	var result []allocationv1.GameServerSelector
	for _, l := range in {
		if c := convertLabelSelectorToInternalLabelSelector(l); c != nil {
			result = append(result, allocationv1.GameServerSelector{LabelSelector: *c})
		}
	}
	return result
}

// ConvertGSAToAllocationResponse converts GameServerAllocation V1 (GSA) to AllocationResponse
func ConvertGSAToAllocationResponse(in *allocationv1.GameServerAllocation) (*pb.AllocationResponse, error) {
	if in == nil {
		return nil, nil
	}

	if err := convertStateV1ToError(in.Status.State); err != nil {
		return nil, err
	}

	return &pb.AllocationResponse{
		GameServerName: in.Status.GameServerName,
		Address:        in.Status.Address,
		NodeName:       in.Status.NodeName,
		Ports:          convertGSAAgonesPortsToAllocationPorts(in.Status.Ports),
	}, nil
}

// ConvertAllocationResponseToGSA converts AllocationResponse to GameServerAllocation V1 (GSA)
func ConvertAllocationResponseToGSA(in *pb.AllocationResponse) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	out := &allocationv1.GameServerAllocation{
		Status: allocationv1.GameServerAllocationStatus{
			State:          allocationv1.GameServerAllocationAllocated,
			GameServerName: in.GameServerName,
			Address:        in.Address,
			NodeName:       in.NodeName,
			Ports:          convertAllocationPortsToGSAAgonesPorts(in.Ports),
		},
	}
	out.SetGroupVersionKind(allocationv1.SchemeGroupVersion.WithKind("GameServerAllocation"))

	return out
}

// convertGSAAgonesPortsToAllocationPorts converts GameServerStatusPort V1 (GSA) to AllocationResponse_GameServerStatusPort
func convertGSAAgonesPortsToAllocationPorts(in []agonesv1.GameServerStatusPort) []*pb.AllocationResponse_GameServerStatusPort {
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

// convertAllocationPortsToGSAAgonesPorts converts AllocationResponse_GameServerStatusPort to GameServerStatusPort V1 (GSA)
func convertAllocationPortsToGSAAgonesPorts(in []*pb.AllocationResponse_GameServerStatusPort) []agonesv1.GameServerStatusPort {
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

// convertStateV1ToError converts GameServerAllocationState V1 (GSA) to AllocationResponse_GameServerAllocationState
func convertStateV1ToError(in allocationv1.GameServerAllocationState) error {
	switch in {
	case allocationv1.GameServerAllocationAllocated:
		return nil
	case allocationv1.GameServerAllocationUnAllocated:
		return status.Error(codes.ResourceExhausted, "there is no available GameServer to allocate")
	case allocationv1.GameServerAllocationContention:
		return status.Error(codes.Aborted, "too many concurrent requests have overwhelmed the system")
	}
	return status.Error(codes.Unknown, "unknown issue")
}
