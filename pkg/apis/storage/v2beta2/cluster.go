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

type MultiRaftClusterState string

const (
	MultiRaftClusterNotReady MultiRaftClusterState = "NotReady"
	MultiRaftClusterReady    MultiRaftClusterState = "Ready"
)

// MultiRaftClusterSpec specifies a MultiRaftClusterSpec configuration
type MultiRaftClusterSpec struct {
	// Replicas is the number of raft replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Groups is the number of groups
	Groups int32 `json:"groups,omitempty"`
	GroupTemplate RaftGroupTemplateSpec `json:"template,omitempty"`
}

type RaftGroupTemplateSpec struct {
	Spec RaftGroupSpec `json:"spec,omitempty"`
}

// MultiRaftClusterStatus defines the status of a RaftCluster
type MultiRaftClusterStatus struct {
	State MultiRaftClusterState `json:"state,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiRaftCluster is the Schema for the RaftCluster API
// +k8s:openapi-gen=true
type MultiRaftCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MultiRaftClusterSpec   `json:"spec,omitempty"`
	Status            MultiRaftClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiRaftClusterList contains a list of MultiRaftCluster
type MultiRaftClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the RaftCluster of items in the list
	Items []MultiRaftCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiRaftCluster{}, &MultiRaftClusterList{})
}