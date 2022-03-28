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

func addMultiRaftProtocolController(mgr manager.Manager) error {
	options := controller.Options{
		Reconciler: &MultiRaftProtocolReconciler{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			events: mgr.GetEventRecorderFor("atomix-raft-storage"),
		},
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*10, time.Second*5),
	}

	// Create a new controller
	controller, err := controller.New("atomix-raft-protocol-v2beta2", mgr, options)
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue Clusters that reference it
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.MultiRaftProtocol{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.MultiRaftCluster{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &storagev2beta2.MultiRaftProtocol{},
		IsController: true,
	})
	if err != nil {
		return err
	}
	return nil
}

// MultiRaftProtocolReconciler reconciles a MultiRaftProtocol object
type MultiRaftProtocolReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	events record.EventRecorder
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *MultiRaftProtocolReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile MultiRaftProtocol")
	protocol := &storagev2beta2.MultiRaftProtocol{}
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

	ok, err = r.reconcileStatus(protocol)
	if err != nil {
		log.Error(err, "Reconcile Cluster")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *MultiRaftProtocolReconciler) reconcileCluster(protocol *storagev2beta2.MultiRaftProtocol) (bool, error) {
	cluster := &storagev2beta2.MultiRaftCluster{}
	clusterName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      protocol.Name,
	}
	if err := r.client.Get(context.TODO(), clusterName, cluster); err != nil {
		if !k8serrors.IsNotFound(err) {
			return false, err
		}
		cluster = &storagev2beta2.MultiRaftCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: protocol.Namespace,
				Name:      protocol.Name,
				Labels:    protocol.Labels,
			},
			Spec: protocol.Spec.MultiRaftClusterSpec,
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

func (r *MultiRaftProtocolReconciler) reconcileStatus(protocol *storagev2beta2.MultiRaftProtocol) (bool, error) {
	cluster := &storagev2beta2.MultiRaftCluster{}
	clusterName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      protocol.Name,
	}
	if err := r.client.Get(context.TODO(), clusterName, cluster); err != nil {
		return false, err
	}

	var state storagev2beta2.MultiRaftProtocolState
	switch cluster.Status.State {
	case storagev2beta2.MultiRaftClusterReady:
		state = storagev2beta2.MultiRaftProtocolReady
	case storagev2beta2.MultiRaftClusterNotReady:
		state = storagev2beta2.MultiRaftProtocolNotReady
	}

	replicas, err := r.getProtocolReplicas(cluster)
	if err != nil {
		return false, err
	}

	partitions, err := r.getProtocolPartitions(cluster)
	if err != nil {
		return false, err
	}

	if protocol.Status.ProtocolStatus == nil ||
		isReplicasChanged(protocol.Status.Replicas, replicas) ||
		isPartitionsChanged(protocol.Status.Partitions, partitions) ||
		protocol.Status.State != state {
		var revision int64
		if protocol.Status.ProtocolStatus != nil {
			revision = protocol.Status.ProtocolStatus.Revision
		}
		if protocol.Status.ProtocolStatus == nil ||
			!isReplicasSame(protocol.Status.Replicas, replicas) ||
			!isPartitionsSame(protocol.Status.Partitions, partitions) {
			revision++
		}

		if state == storagev2beta2.MultiRaftProtocolReady && protocol.Status.State != storagev2beta2.MultiRaftProtocolReady {
			r.events.Eventf(protocol, "Normal", "Ready", "Protocol is ready")
		} else if state == storagev2beta2.MultiRaftProtocolNotReady && protocol.Status.State != storagev2beta2.MultiRaftProtocolNotReady {
			r.events.Eventf(protocol, "Warning", "NotReady", "Protocol is not ready")
		}

		protocol.Status.ProtocolStatus = &corev2beta1.ProtocolStatus{
			Revision:   revision,
			Replicas:   replicas,
			Partitions: partitions,
		}
		protocol.Status.State = state
		if err := r.client.Status().Update(context.TODO(), protocol); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func isReplicasSame(a, b []corev2beta1.ReplicaStatus) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ar := a[i]
		br := b[i]
		if ar.ID != br.ID {
			return false
		}
	}
	return true
}

func isReplicasChanged(a, b []corev2beta1.ReplicaStatus) bool {
	if len(a) != len(b) {
		return true
	}
	for i := 0; i < len(a); i++ {
		ar := a[i]
		br := b[i]
		if ar.ID != br.ID {
			return true
		}
		if ar.Ready != br.Ready {
			return true
		}
	}
	return false
}

func isPartitionsSame(a, b []corev2beta1.PartitionStatus) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ap := a[i]
		bp := b[i]
		if ap.ID != bp.ID {
			return false
		}
		if len(ap.Replicas) != len(bp.Replicas) {
			return false
		}
		for j := 0; j < len(ap.Replicas); j++ {
			if ap.Replicas[j] != bp.Replicas[j] {
				return false
			}
		}
	}
	return true
}

func isPartitionsChanged(a, b []corev2beta1.PartitionStatus) bool {
	if len(a) != len(b) {
		return true
	}
	for i := 0; i < len(a); i++ {
		ap := a[i]
		bp := b[i]
		if ap.ID != bp.ID {
			return true
		}
		if len(ap.Replicas) != len(bp.Replicas) {
			return true
		}
		for j := 0; j < len(ap.Replicas); j++ {
			if ap.Replicas[j] != bp.Replicas[j] {
				return true
			}
		}
		if ap.Ready != bp.Ready {
			return true
		}
	}
	return false
}

func (r *MultiRaftProtocolReconciler) getProtocolReplicas(cluster *storagev2beta2.MultiRaftCluster) ([]corev2beta1.ReplicaStatus, error) {
	numReplicas := getNumReplicas(cluster)
	replicas := make([]corev2beta1.ReplicaStatus, numReplicas)
	for i := 0; i < numReplicas; i++ {
		replicaReady, err := r.isReplicaReady(cluster, i)
		if err != nil {
			return nil, err
		}
		replica := corev2beta1.ReplicaStatus{
			ID:   fmt.Sprintf("%s-%d", cluster.Name, i),
			Host: pointer.StringPtr(getPodDNSName(cluster, i)),
			Port: pointer.Int32Ptr(int32(apiPort)),
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
			Ready: replicaReady,
		}
		replicas[i] = replica
	}
	return replicas, nil
}

func (r *MultiRaftProtocolReconciler) getProtocolPartitions(cluster *storagev2beta2.MultiRaftCluster) ([]corev2beta1.PartitionStatus, error) {
	numReplicas := getNumReplicas(cluster)
	numPartitions := getNumPartitions(cluster)
	partitions := make([]corev2beta1.PartitionStatus, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionID := i + 1
		partitionReady := true
		numMembers := getNumMembers(cluster)
		numROMembers := getNumROMembers(cluster)

		replicas := make([]string, numMembers)
		for j := 0; j < numMembers; j++ {
			replicaID := (((numMembers + numROMembers) * partitionID) + j) % numReplicas
			replicaReady, err := r.isReplicaReady(cluster, replicaID)
			if err != nil {
				return nil, err
			} else if !replicaReady {
				partitionReady = false
			}
			replicas[j] = fmt.Sprintf("%s-%d", cluster.Name, replicaID)
		}

		roReplicas := make([]string, numROMembers)
		for j := 0; j < numROMembers; j++ {
			replicaID := (((numMembers + numROMembers) * partitionID) + numMembers + j) % numReplicas
			replicaReady, err := r.isReplicaReady(cluster, replicaID)
			if err != nil {
				return nil, err
			} else if !replicaReady {
				partitionReady = false
			}
			roReplicas[j] = fmt.Sprintf("%s-%d", cluster.Name, replicaID)
		}
		partition := corev2beta1.PartitionStatus{
			ID:           uint32(partitionID),
			Host:         pointer.StringPtr(fmt.Sprintf("%s.%s", cluster.Name, cluster.Namespace)),
			Port:         pointer.Int32Ptr(apiPort),
			Replicas:     replicas,
			ReadReplicas: roReplicas,
			Ready:        partitionReady,
		}
		partitions[i] = partition
	}
	return partitions, nil
}

func (r *MultiRaftProtocolReconciler) isReplicaReady(cluster *storagev2beta2.MultiRaftCluster, replica int) (bool, error) {
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

var _ reconcile.Reconciler = (*MultiRaftProtocolReconciler)(nil)
