// Copyright 2020-present Open Networking Foundation.
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
	"context"
	"fmt"
	"github.com/atomix/kubernetes-controller/pkg/controller/storage/v2beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	driverTypeEnv      = "ATOMIX_DRIVER_TYPE"
	driverNodeEnv      = "ATOMIX_DRIVER_NODE"
	driverNamespaceEnv = "ATOMIX_DRIVER_NAMESPACE"
	driverNameEnv      = "ATOMIX_DRIVER_NAME"
	driverPortEnv      = "ATOMIX_DRIVER_PORT"
)

const (
	driverName       = "raft"
	driverInjectPath = "/inject-driver"
)

const (
	defaultDriverImage = "atomix/raft-driver:latest"
	defaultDriverPort  = 5681
)

func addDriverController(mgr manager.Manager) error {
	mgr.GetWebhookServer().Register(driverInjectPath, &webhook.Admission{
		Handler: &DriverInjector{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
		},
	})
	return nil
}

type DriverInjector struct {
	client  client.Client
	scheme  *runtime.Scheme
	decoder *admission.Decoder
}

func (i *DriverInjector) InjectDecoder(decoder *admission.Decoder) error {
	i.decoder = decoder
	return nil
}

func (i *DriverInjector) Handle(ctx context.Context, request admission.Request) admission.Response {
	namespacedName := types.NamespacedName{
		Namespace: request.Namespace,
		Name:      request.Name,
	}
	log.Infof("Received admission request for Pod '%s'", namespacedName)

	// Decode the pod
	pod := &corev1.Pod{}
	if err := i.decoder.Decode(request, pod); err != nil {
		log.Errorf("Could not decode Pod '%s'", namespacedName, err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	annotations := v2beta1.NewAnnotations(driverName, pod.Annotations)
	inject, err := annotations.GetInject()
	if err != nil {
		log.Errorf("Driver injection failed for Pod '%s'", namespacedName, err)
		return admission.Errored(http.StatusBadRequest, err)
	} else if !inject {
		log.Infof("Skipping driver injection for Pod '%s'", namespacedName)
		return admission.Allowed("driver injection is not enabled for this pod")
	}

	injected, err := annotations.GetInjected()
	if err != nil {
		log.Errorf("Driver injection failed for Pod '%s'", namespacedName, err)
		return admission.Errored(http.StatusBadRequest, err)
	} else if injected {
		log.Infof("Skipping driver injection for Pod '%s'", namespacedName)
		return admission.Allowed("driver injection is already complete")
	}

	port := defaultDriverPort
	driverPort, err := annotations.GetDriverPort()
	if err != nil {
		log.Errorf("Driver injection failed for Pod '%s'", namespacedName, err)
		return admission.Errored(http.StatusBadRequest, err)
	} else if driverPort != nil {
		port = *driverPort
	}

	image := defaultDriverImage
	driverImage := annotations.GetDriverImage()
	if driverImage != nil {
		image = *driverImage
	}

	container := corev1.Container{
		Name:            "raft-driver",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{
				Name:          "driver",
				ContainerPort: int32(port),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  driverTypeEnv,
				Value: driverName,
			},
			{
				Name:  driverNamespaceEnv,
				Value: namespacedName.Namespace,
			},
			{
				Name:  driverNameEnv,
				Value: namespacedName.Name,
			},
			{
				Name:  driverPortEnv,
				Value: fmt.Sprint(port),
			},
			{
				Name: driverNodeEnv,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		},
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)

	annotations.SetDriverImage(image)
	annotations.SetDriverPort(port)
	annotations.SetInjected(true)

	// Marshal the pod and return a patch response
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Errorf("Driver injection failed for Pod '%s'", namespacedName, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(request.Object.Raw, marshaledPod)
}

var _ admission.Handler = &DriverInjector{}
