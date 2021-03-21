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

package v1beta1

import (
	"context"
	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta3"
	"github.com/atomix/raft-storage-controller/pkg/apis/storage/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Partition ManagementGroup and adds it to the Manager. The Manager will set fields on the ManagementGroup
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
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
	err = c.Watch(&source.Kind{Type: &v1beta1.RaftStorageClass{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: &clusterMapper{
			client: r.client,
		},
	})
	if err != nil {
		return err
	}

	// Watch for changes to referencing resource Database
	err = c.Watch(&source.Kind{Type: &v1beta3.Database{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: &storageFilter{
			client: r.client,
		},
	})
	if err != nil {
		return err
	}

	// Watch for changes to child resource Partition
	err = c.Watch(&source.Kind{Type: &v1beta3.Partition{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1beta3.Database{},
	})
	if err != nil {
		return err
	}
	return nil
}

// clusterMapper is a request mapper that triggers the reconciler for referencing Clusters
// when a RaftStorageClass is changed
type clusterMapper struct {
	client client.Client
}

func (m *clusterMapper) Map(object handler.MapObject) []reconcile.Request {
	log.Info("Filter RaftStorageClass", "Namespace", object.Meta.GetNamespace(), "Name", object.Meta.GetName())

	// Find all databases that reference the changed storage
	databases := &v1beta3.DatabaseList{}
	err := m.client.List(context.TODO(), databases, &client.ListOptions{})
	if err != nil {
		return []reconcile.Request{}
	}

	// Iterate through databases and requeue any that reference the storage controller
	requests := []reconcile.Request{}
	for _, database := range databases.Items {
		if (database.Spec.StorageClass.Group == "" || database.Spec.StorageClass.Group == v1beta1.RaftStorageClassGroup) &&
			(database.Spec.StorageClass.Version == "" || database.Spec.StorageClass.Version == v1beta1.RaftStorageClassVersion) &&
			(database.Spec.StorageClass.Kind == "" || database.Spec.StorageClass.Kind == v1beta1.RaftStorageClassKind) &&
			((database.Spec.StorageClass.Namespace == "" && database.Namespace == object.Meta.GetNamespace()) ||
				(database.Spec.StorageClass.Namespace != "" && database.Spec.StorageClass.Namespace == object.Meta.GetNamespace())) &&
			database.Spec.StorageClass.Name == object.Meta.GetName() {
			log.Info("Matched RaftStorageClass", "Namespace", object.Meta.GetNamespace(), "Name", object.Meta.GetName())
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: database.GetNamespace(),
					Name:      database.GetName(),
				},
			})
		}
	}
	return requests
}

// storageFilter is a request mapper that triggers the reconciler for the storage controller when
// referenced by a watched Cluster
type storageFilter struct {
	client client.Client
}

func (m *storageFilter) Map(object handler.MapObject) []reconcile.Request {
	log.Info("Filter Database", "Namespace", object.Meta.GetNamespace(), "Name", object.Meta.GetName())
	database := object.Object.(*v1beta3.Database)

	// If the Cluster references a RaftStorageClass, enqueue the request
	if (database.Spec.StorageClass.Group == "" || database.Spec.StorageClass.Group == v1beta1.RaftStorageClassGroup) &&
		(database.Spec.StorageClass.Version == "" || database.Spec.StorageClass.Version == v1beta1.RaftStorageClassVersion) &&
		(database.Spec.StorageClass.Kind == "" || database.Spec.StorageClass.Kind == v1beta1.RaftStorageClassKind) {
		storageClass := &v1beta1.RaftStorageClass{}
		namespace := database.Spec.StorageClass.Namespace
		if namespace == "" {
			namespace = database.Namespace
		}
		name := types.NamespacedName{
			Namespace: namespace,
			Name:      database.Spec.StorageClass.Name,
		}
		if err := m.client.Get(context.TODO(), name, storageClass); err != nil {
			log.Error(err, "Filter Database", "Namespace", object.Meta.GetNamespace(), "Name", object.Meta.GetName())
		} else {
			log.Error(err, "Matched Database", "Namespace", object.Meta.GetNamespace(), "Name", object.Meta.GetName())
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: database.GetNamespace(),
						Name:      database.GetName(),
					},
				},
			}
		}
	}
	return []reconcile.Request{}
}
