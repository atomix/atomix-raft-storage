// Copyright 2021-present Open Networking Foundation.
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

package v2beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RaftGroupState string

const (
	RaftGroupNotReady RaftGroupState = "NotReady"
	RaftGroupReady    RaftGroupState = "Ready"
)

// RaftGroupSpec specifies a RaftGroupSpec configuration
type RaftGroupSpec struct {
	GroupID int32 `json:"groupId,omitempty"`

	// Members is the number of members in the group
	Members *int32 `json:"members,omitempty"`

	// ReadOnlyMembers is the number of read-only members in the group
	ReadOnlyMembers *int32 `json:"readOnlyMembers,omitempty"`

	// Image is the image to run
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy to apply
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of secrets for pulling images
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// SecurityContext is a pod security context
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	RaftConfig RaftGroupConfig `json:"raftConfig,omitempty"`

	// VolumeClaimTemplate is the volume claim template for Raft logs
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

type RaftGroupConfig struct {
	HeartbeatPeriod    *metav1.Duration        `json:"heartbeatPeriod,omitempty"`
	ElectionTimeout    *metav1.Duration        `json:"electionTimeout,omitempty"`
	SessionTimeout     *metav1.Duration        `json:"sessionTimeout,omitempty"`
	SnapshotStrategy   *RaftSnapshotStrategy   `json:"snapshotStrategy,omitempty"`
	CompactionStrategy *RaftCompactionStrategy `json:"compactionStrategy,omitempty"`
}

type RaftSnapshotStrategyType string

const (
	RaftSnapshotNone      RaftSnapshotStrategyType = "None"
	RaftSnapshotThreshold RaftSnapshotStrategyType = "Threshold"
)

type RaftSnapshotStrategy struct {
	Type           RaftSnapshotStrategyType `json:"type,omitempty"`
	EntryThreshold *int64                   `json:"entryThreshold,omitempty"`
}

type RaftCompactionStrategyType string

const (
	RaftCompactionNone RaftCompactionStrategyType = "None"
	RaftCompactionAuto RaftCompactionStrategyType = "Auto"
)

type RaftCompactionStrategy struct {
	Type          RaftCompactionStrategyType `json:"type,omitempty"`
	RetainEntries *int64                     `json:"retainEntries,omitempty"`
}

// RaftGroupStatus defines the status of a RaftGroup
type RaftGroupStatus struct {
	State  RaftGroupState `json:"state,omitempty"`
	Leader *string        `json:"leader,omitempty"`
	Term   *uint64        `json:"term,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RaftGroup is the Schema for the RaftGroup API
// +k8s:openapi-gen=true
type RaftGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RaftGroupSpec   `json:"spec,omitempty"`
	Status            RaftGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RaftGroupList contains a list of RaftGroup
type RaftGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the RaftGroup of items in the list
	Items []RaftGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RaftGroup{}, &RaftGroupList{})
}
