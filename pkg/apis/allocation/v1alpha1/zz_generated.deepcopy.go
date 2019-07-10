// +build !ignore_autogenerated

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

// This code was autogenerated. Do not edit directly.

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	agones_v1 "agones.dev/agones/pkg/apis/agones/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GameServerAllocation) DeepCopyInto(out *GameServerAllocation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GameServerAllocation.
func (in *GameServerAllocation) DeepCopy() *GameServerAllocation {
	if in == nil {
		return nil
	}
	out := new(GameServerAllocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GameServerAllocation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GameServerAllocationList) DeepCopyInto(out *GameServerAllocationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GameServerAllocation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GameServerAllocationList.
func (in *GameServerAllocationList) DeepCopy() *GameServerAllocationList {
	if in == nil {
		return nil
	}
	out := new(GameServerAllocationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GameServerAllocationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GameServerAllocationSpec) DeepCopyInto(out *GameServerAllocationSpec) {
	*out = *in
	in.MultiClusterSetting.DeepCopyInto(&out.MultiClusterSetting)
	in.Required.DeepCopyInto(&out.Required)
	if in.Preferred != nil {
		in, out := &in.Preferred, &out.Preferred
		*out = make([]v1.LabelSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.MetaPatch.DeepCopyInto(&out.MetaPatch)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GameServerAllocationSpec.
func (in *GameServerAllocationSpec) DeepCopy() *GameServerAllocationSpec {
	if in == nil {
		return nil
	}
	out := new(GameServerAllocationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GameServerAllocationStatus) DeepCopyInto(out *GameServerAllocationStatus) {
	*out = *in
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]agones_v1.GameServerStatusPort, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GameServerAllocationStatus.
func (in *GameServerAllocationStatus) DeepCopy() *GameServerAllocationStatus {
	if in == nil {
		return nil
	}
	out := new(GameServerAllocationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetaPatch) DeepCopyInto(out *MetaPatch) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetaPatch.
func (in *MetaPatch) DeepCopy() *MetaPatch {
	if in == nil {
		return nil
	}
	out := new(MetaPatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiClusterSetting) DeepCopyInto(out *MultiClusterSetting) {
	*out = *in
	in.PolicySelector.DeepCopyInto(&out.PolicySelector)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiClusterSetting.
func (in *MultiClusterSetting) DeepCopy() *MultiClusterSetting {
	if in == nil {
		return nil
	}
	out := new(MultiClusterSetting)
	in.DeepCopyInto(out)
	return out
}
