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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RaftGroupState is a state constant for RaftGroup
type RaftGroupState string

const (
	// RaftGroupNotReady indicates a RaftGroup is not yet ready
	RaftGroupNotReady RaftGroupState = "NotReady"
	// RaftGroupReady indicates a RaftGroup is ready
	RaftGroupReady RaftGroupState = "Ready"
)

// RaftGroupSpec specifies a RaftGroupSpec configuration
type RaftGroupSpec struct {
	RaftConfig `json:",inline"`
	GroupID    int32 `json:"groupId,omitempty"`
}

// RaftConfig is the configuration of a Raft group
type RaftConfig struct {
	// QuorumSize is the number of replicas in the group
	QuorumSize *int32 `json:"quorumSize,omitempty"`
	// ReadReplicas is the number of read-only replicas in the group
	ReadReplicas            *int32           `json:"readReplicas,omitempty"`
	HeartbeatPeriod         *metav1.Duration `json:"heartbeatPeriod,omitempty"`
	ElectionTimeout         *metav1.Duration `json:"electionTimeout,omitempty"`
	SnapshotEntryThreshold  *int64           `json:"snapshotEntryThreshold,omitempty"`
	CompactionRetainEntries *int64           `json:"compactionRetainEntries,omitempty"`
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
