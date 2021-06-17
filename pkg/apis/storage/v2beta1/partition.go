// Copyright 2019-present Open Networking Foundation.
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

package v2beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RaftPartitionState string

const (
	RaftPartitionNotReady RaftPartitionState = "NotReady"
	RaftPartitionReady    RaftPartitionState = "Ready"
)

// RaftPartitionSpec specifies a RaftPartitionSpec configuration
type RaftPartitionSpec struct {
	ClusterID   int32 `json:"clusterId,omitempty"`
	PartitionID int32 `json:"partitionId,omitempty"`
}

// RaftPartitionStatus defines the status of a RaftPartition
type RaftPartitionStatus struct {
	State  RaftPartitionState `json:"state,omitempty"`
	Leader *string             `json:"leader,omitempty"`
	Term   *uint64             `json:"term,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RaftPartition is the Schema for the RaftPartition API
// +k8s:openapi-gen=true
type RaftPartition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RaftPartitionSpec   `json:"spec,omitempty"`
	Status            RaftPartitionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RaftPartitionList contains a list of RaftPartition
type RaftPartitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the RaftPartition of items in the list
	Items []RaftPartition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RaftPartition{}, &RaftPartitionList{})
}
