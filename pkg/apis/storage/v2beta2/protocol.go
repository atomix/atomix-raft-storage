// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v2beta2

import (
	"github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiRaftProtocolState is a state constant for MultiRaftProtocol
type MultiRaftProtocolState string

const (
	// MultiRaftProtocolNotReady indicates a MultiRaftProtocol is not yet ready
	MultiRaftProtocolNotReady MultiRaftProtocolState = "NotReady"
	// MultiRaftProtocolReady indicates a MultiRaftProtocol is ready
	MultiRaftProtocolReady MultiRaftProtocolState = "Ready"
)

// MultiRaftProtocolSpec specifies a MultiRaftProtocol configuration
type MultiRaftProtocolSpec struct {
	MultiRaftClusterSpec `json:",inline"`
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
