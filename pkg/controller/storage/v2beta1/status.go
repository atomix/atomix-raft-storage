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
	corev2beta1 "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	storagev2beta1 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func (r *Reconciler) reconcileStatus(protocol *storagev2beta1.MultiRaftProtocol) error {
	state := storagev2beta1.MultiRaftProtocolReady
	for _, clusterID := range getClusters(protocol) {
		cluster, err := r.getCluster(protocol, clusterID)
		if err != nil {
			return err
		}
		if err := r.reconcileClusterStatus(protocol, cluster); err != nil {
			return err
		}
		if cluster.Status.State != storagev2beta1.RaftClusterReady {
			state = storagev2beta1.MultiRaftProtocolNotReady
		}
	}

	replicas, err := r.getProtocolReplicas(protocol)
	if err != nil {
		return err
	}

	partitions, err := r.getProtocolPartitions(protocol)
	if err != nil {
		return err
	}

	if protocol.Status.ProtocolStatus == nil ||
		isReplicasChanged(protocol.Status.Replicas, replicas) ||
		isPartitionsChanged(protocol.Status.Partitions, partitions) {
		var revision int64
		if protocol.Status.ProtocolStatus != nil {
			revision = protocol.Status.ProtocolStatus.Revision
		}
		if protocol.Status.ProtocolStatus == nil ||
			!isReplicasSame(protocol.Status.Replicas, replicas) ||
			!isPartitionsSame(protocol.Status.Partitions, partitions) {
			revision++
		}

		if state == storagev2beta1.MultiRaftProtocolReady && protocol.Status.State != storagev2beta1.MultiRaftProtocolReady {
			r.events.Eventf(protocol, "Normal", "Ready", "Protocol is ready")
		} else if state == storagev2beta1.MultiRaftProtocolNotReady && protocol.Status.State != storagev2beta1.MultiRaftProtocolNotReady {
			r.events.Eventf(protocol, "Warning", "NotReady", "Protocol is not ready")
		}

		protocol.Status.ProtocolStatus = &corev2beta1.ProtocolStatus{
			Revision:   revision,
			Replicas:   replicas,
			Partitions: partitions,
		}
		protocol.Status.State = state
		return r.client.Status().Update(context.TODO(), protocol)
	}
	return nil
}

func (r *Reconciler) reconcileClusterStatus(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	status := storagev2beta1.RaftClusterStatus{
		State: storagev2beta1.RaftClusterReady,
	}
	for _, partitionID := range getPartitions(protocol, int(cluster.Spec.ClusterID)) {
		partition, err := r.getPartition(protocol, int(cluster.Spec.ClusterID), partitionID)
		if err != nil {
			return err
		}
		if err := r.reconcilePartitionStatus(protocol, cluster, partition); err != nil {
			return err
		}
		if partition.Status.State != storagev2beta1.RaftPartitionReady {
			status.State = storagev2beta1.RaftClusterNotReady
		}
	}

	if status.State == storagev2beta1.RaftClusterReady && cluster.Status.State != storagev2beta1.RaftClusterReady {
		r.events.Eventf(cluster, "Normal", "Ready", "Cluster is ready")
	} else if status.State == storagev2beta1.RaftClusterNotReady && cluster.Status.State != storagev2beta1.RaftClusterNotReady {
		r.events.Eventf(cluster, "Warning", "NotReady", "Cluster is not ready")
	}
	return r.updateClusterStatus(cluster, status)
}

func (r *Reconciler) reconcilePartitionStatus(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partition *storagev2beta1.RaftPartition) error {
	status := storagev2beta1.RaftPartitionStatus{
		State:  storagev2beta1.RaftPartitionReady,
		Term:   partition.Status.Term,
		Leader: partition.Status.Leader,
	}
	for memberID := range getMembers(protocol, int(cluster.Spec.ClusterID), int(partition.Spec.PartitionID)) {
		member, err := r.getMember(protocol, int(cluster.Spec.ClusterID), int(partition.Spec.PartitionID), memberID)
		if err != nil {
			return err
		}
		if err := r.reconcileMemberStatus(protocol, cluster, partition, member); err != nil {
			return err
		}
		if member.Status.State == nil || *member.Status.State != storagev2beta1.RaftMemberReady {
			status.State = storagev2beta1.RaftPartitionNotReady
		}
		if member.Status.Term != nil && (status.Term == nil || *member.Status.Term > *status.Term) {
			status.Term = member.Status.Term
			status.Leader = nil
			r.events.Eventf(partition, "Normal", "TermChanged", "Term changed to %d", *status.Term)
		}
		if member.Status.Leader != nil && (status.Leader == nil || *status.Leader != *member.Status.Leader) {
			status.Leader = member.Status.Leader
			r.events.Eventf(partition, "Normal", "LeaderChanged", "Leader changed to %s", *status.Leader)
		}
	}

	if status.State == storagev2beta1.RaftPartitionReady && partition.Status.State != storagev2beta1.RaftPartitionReady {
		r.events.Eventf(partition, "Normal", "Ready", "Partition is ready")
	} else if status.State == storagev2beta1.RaftPartitionNotReady && partition.Status.State != storagev2beta1.RaftPartitionNotReady {
		r.events.Eventf(partition, "Warning", "NotReady", "Partition is not ready")
	}
	return r.updatePartitionStatus(partition, status)
}

func (r *Reconciler) reconcileMemberStatus(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partition *storagev2beta1.RaftPartition, member *storagev2beta1.RaftMember) error {
	ready, err := r.isReplicaReady(protocol, int(cluster.Spec.ClusterID), int(member.Spec.MemberID))
	if err != nil {
		return err
	}

	if ready && (member.Status.State == nil || *member.Status.State != storagev2beta1.RaftMemberReady) {
		state := storagev2beta1.RaftMemberReady
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			State: &state,
		})
	} else if !ready && (member.Status.State == nil || *member.Status.State != storagev2beta1.RaftMemberNotReady) {
		state := storagev2beta1.RaftMemberNotReady
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			State: &state,
		})
	}
	return nil
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

func (r *Reconciler) getProtocolReplicas(protocol *storagev2beta1.MultiRaftProtocol) ([]corev2beta1.ReplicaStatus, error) {
	replicas := make([]corev2beta1.ReplicaStatus, 0, getNumReplicas(protocol)*getNumClusters(protocol))
	for _, clusterID := range getClusters(protocol) {
		for replicaID, replicaName := range getReplicas(protocol, clusterID) {
			replicaReady, err := r.isReplicaReady(protocol, clusterID, replicaID)
			if err != nil {
				return nil, err
			}
			replica := corev2beta1.ReplicaStatus{
				ID:   replicaName,
				Host: pointer.StringPtr(getPodDNSName(protocol, clusterID, replicaID)),
				Port: pointer.Int32Ptr(int32(apiPort)),
				ExtraPorts: map[string]int32{
					protocolPortName: protocolPort,
				},
				Ready: replicaReady,
			}
			replicas = append(replicas, replica)
		}
	}
	return replicas, nil
}

func (r *Reconciler) getProtocolPartitions(protocol *storagev2beta1.MultiRaftProtocol) ([]corev2beta1.PartitionStatus, error) {
	partitions := make([]corev2beta1.PartitionStatus, 0, protocol.Spec.Partitions)
	for _, clusterID := range getClusters(protocol) {
		for _, partitionID := range getPartitions(protocol, clusterID) {
			partitionReady := true
			for replicaID := range getReplicas(protocol, clusterID) {
				replicaReady, err := r.isReplicaReady(protocol, clusterID, replicaID)
				if err != nil {
					return nil, err
				} else if !replicaReady {
					partitionReady = false
				}
			}
			partitions = append(partitions, corev2beta1.PartitionStatus{
				ID:       uint32(partitionID),
				Replicas: getReplicas(protocol, clusterID),
				Ready:    partitionReady,
			})
		}
	}
	return partitions, nil
}

func (r *Reconciler) isReplicaReady(protocol *storagev2beta1.MultiRaftProtocol, cluster int, replica int) (bool, error) {
	podName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getPodName(protocol, cluster, replica),
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

func (r *Reconciler) updateClusterStatus(cluster *storagev2beta1.RaftCluster, status storagev2beta1.RaftClusterStatus) error {
	updated := true
	if cluster.Status.State != status.State {
		cluster.Status.State = status.State
		updated = true
	}
	if updated {
		return r.client.Status().Update(context.TODO(), cluster)
	}
	return nil
}

func (r *Reconciler) updatePartitionStatus(partition *storagev2beta1.RaftPartition, status storagev2beta1.RaftPartitionStatus) error {
	updated := true
	if status.Term != nil && (partition.Status.Term == nil || *status.Term > *partition.Status.Term) {
		partition.Status.Term = status.Term
		partition.Status.Leader = nil
		updated = true
	}
	if status.Leader != nil && (partition.Status.Leader == nil || partition.Status.Leader != status.Leader) {
		partition.Status.Leader = status.Leader
		updated = true
	}
	if partition.Status.State != status.State {
		partition.Status.State = status.State
		updated = true
	}
	if updated {
		return r.client.Status().Update(context.TODO(), partition)
	}
	return nil
}

func (r *Reconciler) updateMemberStatus(member *storagev2beta1.RaftMember, status storagev2beta1.RaftMemberStatus) error {
	updated := true
	if status.Term != nil && (member.Status.Term == nil || *status.Term > *member.Status.Term) {
		member.Status.Term = status.Term
		member.Status.Leader = nil
		updated = true
	}
	if status.Leader != nil && (member.Status.Leader == nil || member.Status.Leader != status.Leader) {
		member.Status.Leader = status.Leader
		updated = true
	}
	if status.State != nil && (member.Status.State == nil || member.Status.State != status.State) {
		member.Status.State = status.State
		updated = true
	}
	if status.Role != nil && (member.Status.Role == nil || *member.Status.Role != *status.Role) {
		member.Status.Role = status.Role
		updated = true
	}
	if status.LastUpdated != nil && (member.Status.LastUpdated == nil || status.LastUpdated.After(member.Status.LastUpdated.Time)) {
		member.Status.LastUpdated = status.LastUpdated
		updated = true
	}
	if status.LastSnapshotIndex != nil && (member.Status.LastSnapshotIndex == nil || *status.LastSnapshotIndex > *member.Status.LastSnapshotIndex) {
		member.Status.LastSnapshotIndex = status.LastSnapshotIndex
		updated = true
	}
	if status.LastSnapshotTime != nil && (member.Status.LastSnapshotTime == nil || status.LastSnapshotTime.After(member.Status.LastSnapshotTime.Time)) {
		member.Status.LastSnapshotTime = status.LastSnapshotTime
		updated = true
	}
	if updated {
		return r.client.Status().Update(context.TODO(), member)
	}
	return nil
}
