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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
	"time"

	storagev2beta1 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func addMultiRaftProtocolController(mgr manager.Manager) error {
	options := controller.Options{
		Reconciler: &MultiRaftProtocolReconciler{
			client:  mgr.GetClient(),
			scheme:  mgr.GetScheme(),
			events:  mgr.GetEventRecorderFor("atomix-raft-storage"),
			streams: make(map[string]func()),
		},
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*10, time.Second*5),
	}

	// Create a new controller
	controller, err := controller.New(mgr.GetScheme().Name(), mgr, options)
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue Clusters that reference it
	err = controller.Watch(&source.Kind{Type: &storagev2beta1.MultiRaftProtocol{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet
	err = controller.Watch(&source.Kind{Type: &storagev2beta1.MultiRaftCluster{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &storagev2beta1.MultiRaftProtocol{},
		IsController: true,
	})
	if err != nil {
		return err
	}
	return nil
}

// MultiRaftProtocolReconciler reconciles a MultiRaftProtocol object
type MultiRaftProtocolReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	events  record.EventRecorder
	streams map[string]func()
	mu      sync.Mutex
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *MultiRaftProtocolReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile MultiRaftProtocol")
	protocol := &storagev2beta1.MultiRaftProtocol{}
	err := r.client.Get(context.TODO(), request.NamespacedName, protocol)
	if err != nil {
		log.Error(err, "Reconcile MultiRaftProtocol")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ok, err := r.reconcileCluster(protocol)
	if err != nil {
		log.Error(err, "Reconcile Cluster")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *MultiRaftProtocolReconciler) reconcileCluster(protocol *storagev2beta1.MultiRaftProtocol) (bool, error) {
	cluster := &storagev2beta1.MultiRaftCluster{}
	clusterName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      protocol.Name,
	}
	if err := r.client.Get(context.TODO(), clusterName, cluster); err != nil {
		if !k8serrors.IsNotFound(err) {
			return false, err
		}
		cluster = &storagev2beta1.MultiRaftCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: protocol.Namespace,
				Name:      protocol.Name,
				Labels:    protocol.Labels,
			},
			Spec: protocol.Spec.Cluster,
		}
		if err := controllerutil.SetControllerReference(protocol, cluster, r.scheme); err != nil {
			return false, err
		}
		if err := r.client.Create(context.TODO(), cluster); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

var _ reconcile.Reconciler = (*MultiRaftProtocolReconciler)(nil)
