// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v2beta2

import (
	"context"
	"fmt"
	corev2beta1 "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			events: mgr.GetEventRecorderFor("atomix-raft-storage"),
		},
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*10, time.Second*5),
	}

	// Create a new controller
	controller, err := controller.New("atomix-raft-group-v2beta2", mgr, options)
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
	client client.Client
	scheme *runtime.Scheme
	events record.EventRecorder
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

	ok, err = r.reconcileStatus(cluster, group)
	if err != nil {
		log.Error(err, "Reconcile Cluster")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *RaftGroupReconciler) reconcileMembers(cluster *storagev2beta2.MultiRaftCluster, group *storagev2beta2.RaftGroup) (bool, error) {
	for i := 0; i < getNumMembers(cluster); i++ {
		if ok, err := r.reconcileMember(cluster, group, i, false); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}

	for i := 0; i < getNumROMembers(cluster); i++ {
		if ok, err := r.reconcileMember(cluster, group, getNumMembers(cluster)+i, true); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

func (r *RaftGroupReconciler) reconcileMember(cluster *storagev2beta2.MultiRaftCluster, group *storagev2beta2.RaftGroup, memberID int, readOnly bool) (bool, error) {
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
			Spec: storagev2beta2.RaftMemberSpec{
				PodName:  fmt.Sprintf("%s-%d", cluster.Name, ((getNumMembers(cluster)*int(group.Spec.GroupID))+memberID)%getNumReplicas(cluster)),
				ReadOnly: readOnly,
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

func (r *RaftGroupReconciler) reconcileStatus(cluster *storagev2beta2.MultiRaftCluster, group *storagev2beta2.RaftGroup) (bool, error) {
	replicas, err := r.getReplicas(cluster, group)
	if err != nil {
		return false, err
	}

	changed := false
	state := storagev2beta2.RaftGroupReady
	for _, replica := range replicas {
		if !replica.Ready {
			state = storagev2beta2.RaftGroupNotReady
			break
		}
	}

	if group.Status.State != state {
		group.Status.State = state
		r.events.Event(group, "Normal", string(state), "Group state changed")
		changed = true
	}

	for memberID := 0; memberID < getNumMembers(cluster); memberID++ {
		member := &storagev2beta2.RaftMember{}
		memberName := types.NamespacedName{
			Namespace: group.Namespace,
			Name:      fmt.Sprintf("%s-%d", group.Name, memberID),
		}
		if err := r.client.Get(context.TODO(), memberName, member); err != nil {
			return false, err
		}

		if member.Status.Term != nil && (group.Status.Term == nil || *member.Status.Term > *group.Status.Term) {
			group.Status.Term = member.Status.Term
			group.Status.Leader = nil
			r.events.Eventf(group, "Normal", "TermChanged", "Term changed to %d", *group.Status.Term)
			changed = true
		}

		if member.Status.Leader != nil && group.Status.Leader == nil {
			group.Status.Leader = member.Status.Leader
			r.events.Eventf(group, "Normal", "LeaderChanged", "Leader changed to %s", *group.Status.Leader)
			changed = true
		}
	}

	if changed {
		if err := r.client.Status().Update(context.TODO(), group); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *RaftGroupReconciler) getReplicas(cluster *storagev2beta2.MultiRaftCluster, group *storagev2beta2.RaftGroup) ([]corev2beta1.ReplicaStatus, error) {
	numReplicas := getNumReplicas(cluster)
	partitionID := group.Spec.GroupID
	numMembers := getNumMembers(cluster)
	numROMembers := getNumROMembers(cluster)

	replicas := make([]corev2beta1.ReplicaStatus, 0, numReplicas)
	for i := 0; i < numMembers; i++ {
		replicaID := (((numMembers + numROMembers) * int(partitionID)) + i) % numReplicas
		replicaReady, err := r.isReplicaReady(cluster, replicaID)
		if err != nil {
			return nil, err
		}
		replica := corev2beta1.ReplicaStatus{
			ID:   fmt.Sprintf("%s-%d", cluster.Name, replicaID),
			Host: pointer.StringPtr(getPodDNSName(cluster, replicaID)),
			Port: pointer.Int32Ptr(int32(apiPort)),
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
			Ready: replicaReady,
		}
		replicas = append(replicas, replica)
	}

	for i := 0; i < numROMembers; i++ {
		replicaID := (((numMembers + numROMembers) * int(partitionID)) + numMembers + i) % numReplicas
		replicaReady, err := r.isReplicaReady(cluster, replicaID)
		if err != nil {
			return nil, err
		}
		replica := corev2beta1.ReplicaStatus{
			ID:   fmt.Sprintf("%s-%d", cluster.Name, replicaID),
			Host: pointer.StringPtr(getPodDNSName(cluster, replicaID)),
			Port: pointer.Int32Ptr(int32(apiPort)),
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
			Ready: replicaReady,
		}
		replicas = append(replicas, replica)
	}
	return replicas, nil
}

func (r *RaftGroupReconciler) isReplicaReady(cluster *storagev2beta2.MultiRaftCluster, replica int) (bool, error) {
	podName := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      fmt.Sprintf("%s-%d", cluster.Name, replica),
	}
	pod := &corev1.Pod{}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue, nil
		}
	}
	return false, nil
}

var _ reconcile.Reconciler = (*RaftGroupReconciler)(nil)
