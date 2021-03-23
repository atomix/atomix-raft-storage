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
	"github.com/atomix/raft-storage-controller/pkg/apis/storage/v2beta1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AddControllers creates a new Partition ManagementGroup and adds it to the Manager. The Manager will set fields on the ManagementGroup
// and Start it when the Manager is Started.
func AddControllers(mgr manager.Manager) error {
	log.Info("Add manager")
	r := &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}

	// Create a new controller
	c, err := controller.New(mgr.GetScheme().Name(), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue Clusters that reference it
	err = c.Watch(&source.Kind{Type: &v2beta1.RaftProtocol{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v2beta1.RaftProtocol{},
		IsController: false,
	 })
	if err != nil {
		return err
	}
	return nil
}
