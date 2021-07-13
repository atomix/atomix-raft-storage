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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftCluster) DeepCopyInto(out *MultiRaftCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftCluster.
func (in *MultiRaftCluster) DeepCopy() *MultiRaftCluster {
	if in == nil {
		return nil
	}
	out := new(MultiRaftCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRaftCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftClusterList) DeepCopyInto(out *MultiRaftClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MultiRaftCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftClusterList.
func (in *MultiRaftClusterList) DeepCopy() *MultiRaftClusterList {
	if in == nil {
		return nil
	}
	out := new(MultiRaftClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MultiRaftClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftClusterSpec) DeepCopyInto(out *MultiRaftClusterSpec) {
	*out = *in
	in.GroupTemplate.DeepCopyInto(&out.GroupTemplate)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftClusterSpec.
func (in *MultiRaftClusterSpec) DeepCopy() *MultiRaftClusterSpec {
	if in == nil {
		return nil
	}
	out := new(MultiRaftClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultiRaftClusterStatus) DeepCopyInto(out *MultiRaftClusterStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultiRaftClusterStatus.
func (in *MultiRaftClusterStatus) DeepCopy() *MultiRaftClusterStatus {
	if in == nil {
		return nil
	}
	out := new(MultiRaftClusterStatus)
	in.DeepCopyInto(out)
	return out
}

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
	in.Cluster.DeepCopyInto(&out.Cluster)
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftCompactionStrategy) DeepCopyInto(out *RaftCompactionStrategy) {
	*out = *in
	if in.RetainEntries != nil {
		in, out := &in.RetainEntries, &out.RetainEntries
		*out = new(int64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftCompactionStrategy.
func (in *RaftCompactionStrategy) DeepCopy() *RaftCompactionStrategy {
	if in == nil {
		return nil
	}
	out := new(RaftCompactionStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroup) DeepCopyInto(out *RaftGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroup.
func (in *RaftGroup) DeepCopy() *RaftGroup {
	if in == nil {
		return nil
	}
	out := new(RaftGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroupConfig) DeepCopyInto(out *RaftGroupConfig) {
	*out = *in
	if in.HeartbeatPeriod != nil {
		in, out := &in.HeartbeatPeriod, &out.HeartbeatPeriod
		*out = new(v1.Duration)
		**out = **in
	}
	if in.ElectionTimeout != nil {
		in, out := &in.ElectionTimeout, &out.ElectionTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.SessionTimeout != nil {
		in, out := &in.SessionTimeout, &out.SessionTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.SnapshotStrategy != nil {
		in, out := &in.SnapshotStrategy, &out.SnapshotStrategy
		*out = new(RaftSnapshotStrategy)
		(*in).DeepCopyInto(*out)
	}
	if in.CompactionStrategy != nil {
		in, out := &in.CompactionStrategy, &out.CompactionStrategy
		*out = new(RaftCompactionStrategy)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroupConfig.
func (in *RaftGroupConfig) DeepCopy() *RaftGroupConfig {
	if in == nil {
		return nil
	}
	out := new(RaftGroupConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroupList) DeepCopyInto(out *RaftGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RaftGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroupList.
func (in *RaftGroupList) DeepCopy() *RaftGroupList {
	if in == nil {
		return nil
	}
	out := new(RaftGroupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroupSpec) DeepCopyInto(out *RaftGroupSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]corev1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(corev1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	in.RaftConfig.DeepCopyInto(&out.RaftConfig)
	if in.VolumeClaimTemplate != nil {
		in, out := &in.VolumeClaimTemplate, &out.VolumeClaimTemplate
		*out = new(corev1.PersistentVolumeClaim)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroupSpec.
func (in *RaftGroupSpec) DeepCopy() *RaftGroupSpec {
	if in == nil {
		return nil
	}
	out := new(RaftGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroupStatus) DeepCopyInto(out *RaftGroupStatus) {
	*out = *in
	if in.Leader != nil {
		in, out := &in.Leader, &out.Leader
		*out = new(string)
		**out = **in
	}
	if in.Term != nil {
		in, out := &in.Term, &out.Term
		*out = new(uint64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroupStatus.
func (in *RaftGroupStatus) DeepCopy() *RaftGroupStatus {
	if in == nil {
		return nil
	}
	out := new(RaftGroupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftGroupTemplateSpec) DeepCopyInto(out *RaftGroupTemplateSpec) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftGroupTemplateSpec.
func (in *RaftGroupTemplateSpec) DeepCopy() *RaftGroupTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(RaftGroupTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftMember) DeepCopyInto(out *RaftMember) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftMember.
func (in *RaftMember) DeepCopy() *RaftMember {
	if in == nil {
		return nil
	}
	out := new(RaftMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftMember) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftMemberList) DeepCopyInto(out *RaftMemberList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RaftMember, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftMemberList.
func (in *RaftMemberList) DeepCopy() *RaftMemberList {
	if in == nil {
		return nil
	}
	out := new(RaftMemberList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftMemberList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftMemberSpec) DeepCopyInto(out *RaftMemberSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftMemberSpec.
func (in *RaftMemberSpec) DeepCopy() *RaftMemberSpec {
	if in == nil {
		return nil
	}
	out := new(RaftMemberSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftMemberStatus) DeepCopyInto(out *RaftMemberStatus) {
	*out = *in
	if in.State != nil {
		in, out := &in.State, &out.State
		*out = new(RaftMemberState)
		**out = **in
	}
	if in.Role != nil {
		in, out := &in.Role, &out.Role
		*out = new(RaftMemberRole)
		**out = **in
	}
	if in.Leader != nil {
		in, out := &in.Leader, &out.Leader
		*out = new(string)
		**out = **in
	}
	if in.Term != nil {
		in, out := &in.Term, &out.Term
		*out = new(uint64)
		**out = **in
	}
	if in.LastUpdated != nil {
		in, out := &in.LastUpdated, &out.LastUpdated
		*out = (*in).DeepCopy()
	}
	if in.LastSnapshotIndex != nil {
		in, out := &in.LastSnapshotIndex, &out.LastSnapshotIndex
		*out = new(uint64)
		**out = **in
	}
	if in.LastSnapshotTime != nil {
		in, out := &in.LastSnapshotTime, &out.LastSnapshotTime
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftMemberStatus.
func (in *RaftMemberStatus) DeepCopy() *RaftMemberStatus {
	if in == nil {
		return nil
	}
	out := new(RaftMemberStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftSessionConfig) DeepCopyInto(out *RaftSessionConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftSessionConfig.
func (in *RaftSessionConfig) DeepCopy() *RaftSessionConfig {
	if in == nil {
		return nil
	}
	out := new(RaftSessionConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftSessionConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftSessionConfigList) DeepCopyInto(out *RaftSessionConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RaftSessionConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftSessionConfigList.
func (in *RaftSessionConfigList) DeepCopy() *RaftSessionConfigList {
	if in == nil {
		return nil
	}
	out := new(RaftSessionConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RaftSessionConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RaftSnapshotStrategy) DeepCopyInto(out *RaftSnapshotStrategy) {
	*out = *in
	if in.EntryThreshold != nil {
		in, out := &in.EntryThreshold, &out.EntryThreshold
		*out = new(int64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RaftSnapshotStrategy.
func (in *RaftSnapshotStrategy) DeepCopy() *RaftSnapshotStrategy {
	if in == nil {
		return nil
	}
	out := new(RaftSnapshotStrategy)
	in.DeepCopyInto(out)
	return out
}
