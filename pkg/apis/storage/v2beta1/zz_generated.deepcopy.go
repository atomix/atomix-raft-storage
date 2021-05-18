// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v2beta1

import (
	corev2beta1 "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftProtocol) DeepCopyInto(out *MultiRaftProtocol) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftProtocol.
func (in *MultiRaftProtocol) DeepCopy() *MultiRaftProtocol {
	if in == nil {
		return nil
	}
	out := new(MultiRaftProtocol)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRaftProtocol) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftProtocolList) DeepCopyInto(out *MultiRaftProtocolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MultiRaftProtocol, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftProtocolList.
func (in *MultiRaftProtocolList) DeepCopy() *MultiRaftProtocolList {
	if in == nil {
		return nil
	}
	out := new(MultiRaftProtocolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRaftProtocolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftProtocolSpec) DeepCopyInto(out *MultiRaftProtocolSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.VolumeClaimTemplate != nil {
		in, out := &in.VolumeClaimTemplate, &out.VolumeClaimTemplate
		*out = new(v1.PersistentVolumeClaim)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftProtocolSpec.
func (in *MultiRaftProtocolSpec) DeepCopy() *MultiRaftProtocolSpec {
	if in == nil {
		return nil
	}
	out := new(MultiRaftProtocolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftProtocolStatus) DeepCopyInto(out *MultiRaftProtocolStatus) {
	*out = *in
	if in.ProtocolStatus != nil {
		in, out := &in.ProtocolStatus, &out.ProtocolStatus
		*out = new(corev2beta1.ProtocolStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftProtocolStatus.
func (in *MultiRaftProtocolStatus) DeepCopy() *MultiRaftProtocolStatus {
	if in == nil {
		return nil
	}
	out := new(MultiRaftProtocolStatus)
	in.DeepCopyInto(out)
	return out
}