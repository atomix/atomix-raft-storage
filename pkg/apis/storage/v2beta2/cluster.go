// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v2beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiRaftClusterState is a state constant for MultiRaftCluster
type MultiRaftClusterState string

const (
	// MultiRaftClusterNotReady indicates a MultiRaftCluster is not yet ready
	MultiRaftClusterNotReady MultiRaftClusterState = "NotReady"
	// MultiRaftClusterReady indicates a MultiRaftCluster is ready
	MultiRaftClusterReady MultiRaftClusterState = "Ready"
)

// MultiRaftClusterSpec specifies a MultiRaftClusterSpec configuration
type MultiRaftClusterSpec struct {
	// Replicas is the number of raft replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Groups is the number of groups
	Groups int32 `json:"groups,omitempty"`

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

	// Raft is the Raft protocol configuration
	Raft RaftConfig `json:"raft,omitempty"`
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
