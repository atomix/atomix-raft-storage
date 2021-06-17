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
	"github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MultiRaftProtocolState string

const (
	MultiRaftProtocolNotReady MultiRaftProtocolState = "NotReady"
	MultiRaftProtocolReady    MultiRaftProtocolState = "Ready"
)

// MultiRaftProtocolSpec specifies a MultiRaftProtocol configuration
type MultiRaftProtocolSpec struct {
	// Clusters is the number of clusters to create
	Clusters int32 `json:"clusters,omitempty"`

	// Partitions is the number of partitions
	Partitions int32 `json:"partitions,omitempty"`

	// Replicas is the number of raft replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the image to run
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy to apply
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of secrets for pulling images
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// SecurityContext is a pod security context
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// VolumeClaimTemplate is the volume claim template for Raft logs
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
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
