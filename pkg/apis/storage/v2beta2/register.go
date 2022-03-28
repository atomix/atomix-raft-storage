// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// NOTE: Boilerplate only.  Ignore this file.

// Package v2beta2 contains API Schema definitions for the cloud v2beta1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=storage.atomix.io
package v2beta2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "storage.atomix.io", Version: "v2beta2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme is required by the client code generator
	AddToScheme = SchemeBuilder.AddToScheme
)
