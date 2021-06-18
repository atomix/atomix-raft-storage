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
	storagev2beta1 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta1"
	"github.com/atomix/atomix-raft-storage/pkg/storage"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) recordPartitionReady(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.MemberReadyEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "MemberReady", "Member is ready to receive requests on partition %d", event.Partition)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "Ready", "Member is ready to receive requests")

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		state := storagev2beta1.RaftMemberReady
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			State:       &state,
			LastUpdated: &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *Reconciler) recordLeaderUpdated(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.LeaderUpdatedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}

	if member.Status.Term == nil || *member.Status.Term != event.Term {
		r.events.Eventf(member, "Normal", "TermChanged", "Term changed to %d", event.Term)
		r.events.Eventf(pod, "Normal", "PartitionTermChanged", "Term for partition %d changed to %d", event.Partition, event.Term)
	}

	if event.Leader != "" && (member.Status.Leader == nil || *member.Status.Leader != event.Leader) {
		r.events.Eventf(member, "Normal", "LeaderChanged", "Leader for term %d changed to %s", event.Term, event.Leader)
		r.events.Eventf(pod, "Normal", "PartitionLeaderChanged", "Leader for partition %d changed to %s for term %d", event.Partition, event.Leader, event.Term)
	}

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		role := storagev2beta1.RaftFollower
		if event.Leader == getPodName(protocol, clusterID, podID) {
			role = storagev2beta1.RaftLeader
		}
		var leader *string
		if event.Leader != "" {
			leader = &event.Leader
		}
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			Role:        &role,
			Term:        &event.Term,
			Leader:      leader,
			LastUpdated: &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *Reconciler) recordMembershipChanged(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.MembershipChangedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionMembershipChanged", "Membership changed in partition %d", event.Partition)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "MembershipChanged", "Membership changed")
}

func (r *Reconciler) recordSendSnapshotStarted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SendSnapshotStartedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSendSnapshotStared", "Started sending partition %d snapshot at index %d to %s", event.Partition, event.Index, event.To)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SendSnapshotStared", "Started sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *Reconciler) recordSendSnapshotCompleted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SendSnapshotCompletedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSendSnapshotCompleted", "Completed sending partition %d snapshot at index %d to %s", event.Partition, event.Index, event.To)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SendSnapshotCompleted", "Completed sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *Reconciler) recordSendSnapshotAborted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SendSnapshotAbortedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Warning", "PartitionSendSnapshotAborted", "Aborted sending partition %d snapshot at index %d to %s", event.Partition, event.Index, event.To)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Warning", "SendSnapshotAborted", "Aborted sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *Reconciler) recordSnapshotReceived(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SnapshotReceivedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSnapshotReceived", "Partition %d received snapshot at index %d from %s", event.Partition, event.Index, event.From)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SnapshotReceived", "Received snapshot at index %d from %s", event.Index, event.From)

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
}

func (r *Reconciler) recordSnapshotRecovered(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SnapshotRecoveredEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSnapshotRecovered", "Recovered partition %d from snapshot at index %d", event.Partition, event.Index)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SnapshotRecovered", "Recovered from snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
}

func (r *Reconciler) recordSnapshotCreated(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SnapshotCreatedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSnapshotCreated", "Created partition %d snapshot at index %d", event.Partition, event.Index)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SnapshotCreated", "Created snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
}

func (r *Reconciler) recordSnapshotCompacted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.SnapshotCompactedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionSnapshotCompacted", "Compacted partition %d snapshot at index %d", event.Partition, event.Index)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "SnapshotCompacted", "Compacted snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta1.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
}

func (r *Reconciler) recordLogCompacted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.LogCompactedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionLogCompacted", "Log in partition %d compacted at index %d", event.Partition, event.Index)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "LogCompacted", "Log compacted at index %d", event.Index)
}

func (r *Reconciler) recordLogDBCompacted(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.LogDBCompactedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "PartitionLogDBCompacted", "LogDB in partition %d compacted at index %d", event.Partition, event.Index)

	member, err := r.getMember(protocol, clusterID, int(event.Partition), podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(member, "Normal", "LogDBCompacted", "LogDB compacted at index %d", event.Index)
}

func (r *Reconciler) recordConnectionEstablished(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.ConnectionEstablishedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "ConnectionEstablished", "Connection established to %s", event.Address)
}

func (r *Reconciler) recordConnectionFailed(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int, event *storage.ConnectionFailedEvent, timestamp metav1.Time) {
	pod, err := r.getPod(protocol, clusterID, podID)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Warning", "ConnectionFailed", "Connection to %s failed", event.Address)
}
