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
	"context"
	"fmt"
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

	storagev2beta2 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func addRaftGroupController(mgr manager.Manager) error {
	options := controller.Options{
		Reconciler: &RaftGroupReconciler{
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
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.RaftGroup{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource RaftMember
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.RaftMember{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &storagev2beta2.RaftGroup{},
		IsController: true,
	})
	if err != nil {
		return err
	}
	return nil
}

// RaftGroupReconciler reconciles a RaftGroup object
type RaftGroupReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	events  record.EventRecorder
	streams map[string]func()
	mu      sync.Mutex
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *RaftGroupReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile RaftGroup")
	group := &storagev2beta2.RaftGroup{}
	err := r.client.Get(context.TODO(), request.NamespacedName, group)
	if err != nil {
		log.Error(err, "Reconcile RaftGroup")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var cluster *storagev2beta2.MultiRaftCluster
	for _, ref := range group.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			clusterName := types.NamespacedName{
				Namespace: group.Namespace,
				Name:      ref.Name,
			}
			cluster = &storagev2beta2.MultiRaftCluster{}
			if err := r.client.Get(context.TODO(), clusterName, cluster); err != nil {
				return reconcile.Result{}, err
			}
			break
		}
	}

	ok, err := r.reconcileMembers(cluster, group)
	if err != nil {
		log.Error(err, "Reconcile RaftGroup")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *RaftGroupReconciler) reconcileMembers(cluster *storagev2beta2.MultiRaftCluster, group *storagev2beta2.RaftGroup) (bool, error) {
	replicas := int(cluster.Spec.Replicas)
	if group.Spec.Members != nil {
		replicas = int(*group.Spec.Members)
	}
	for i := 0; i < replicas; i++ {
		if ok, err := r.reconcileMember(group, i, false); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}

	readReplicas := 0
	if group.Spec.ReadOnlyMembers != nil {
		readReplicas = int(*group.Spec.ReadOnlyMembers)
	}
	for i := 0; i < readReplicas; i++ {
		if ok, err := r.reconcileMember(group, replicas+i, true); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

func (r *RaftGroupReconciler) reconcileMember(group *storagev2beta2.RaftGroup, memberID int, readOnly bool) (bool, error) {
	member := &storagev2beta2.RaftMember{}
	memberName := types.NamespacedName{
		Namespace: group.Namespace,
		Name:      fmt.Sprintf("%s-%d", group.Name, memberID),
	}
	if err := r.client.Get(context.TODO(), memberName, member); err != nil {
		if !k8serrors.IsNotFound(err) {
			return false, err
		}
		member = &storagev2beta2.RaftMember{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: memberName.Namespace,
				Name:      memberName.Name,
				Labels:    group.Labels,
			},
		}
		if err := controllerutil.SetControllerReference(group, member, r.scheme); err != nil {
			return false, err
		}
		if err := r.client.Create(context.TODO(), member); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

var _ reconcile.Reconciler = (*RaftGroupReconciler)(nil)