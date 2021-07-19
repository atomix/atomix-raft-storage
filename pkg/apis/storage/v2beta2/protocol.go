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
	"github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MultiRaftProtocolState string

const (
	MultiRaftProtocolNotReady MultiRaftProtocolState = "NotReady"
	MultiRaftProtocolReady    MultiRaftProtocolState = "Ready"
)

// MultiRaftProtocolSpec specifies a MultiRaftProtocol configuration
type MultiRaftProtocolSpec struct {
	Cluster MultiRaftClusterSpec `json:"cluster,omitempty"`
}

// MultiRaftProtocolStatus defines the status of a MultiRaftProtocol
type MultiRaftProtocolStatus struct {
	*v2beta1.ProtocolStatus `json:",inline"`
	State                   MultiRaftProtocolState `json:"state,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiRaftProtocol is the Schema for the MultiRaftProtocol API
// +k8s:openapi-gen=true
type MultiRaftProtocol struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MultiRaftProtocolSpec   `json:"spec,omitempty"`
	Status            MultiRaftProtocolStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiRaftProtocolList contains a list of MultiRaftProtocol
type MultiRaftProtocolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the MultiRaftProtocol of items in the list
	Items []MultiRaftProtocol `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiRaftProtocol{}, &MultiRaftProtocolList{})
}