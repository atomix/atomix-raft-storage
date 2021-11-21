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
	"errors"
	"fmt"
	"github.com/atomix/atomix-raft-storage/pkg/engine"
	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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

func addRaftMemberController(mgr manager.Manager) error {
	options := controller.Options{
		Reconciler: &RaftMemberReconciler{
			client:  mgr.GetClient(),
			scheme:  mgr.GetScheme(),
			events:  mgr.GetEventRecorderFor("atomix-raft-storage"),
			streams: make(map[string]func()),
		},
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*10, time.Second*5),
	}

	// Create a new controller
	controller, err := controller.New("atomix-raft-member-v2beta2", mgr, options)
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue RaftMembers that reference it
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.RaftMember{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = controller.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: &mapPodMember{
			client: mgr.GetClient(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// RaftMemberReconciler reconciles a RaftMember object
type RaftMemberReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	events  record.EventRecorder
	streams map[string]func()
	mu      sync.Mutex
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *RaftMemberReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile RaftMember")
	member := &storagev2beta2.RaftMember{}
	err := r.client.Get(context.TODO(), request.NamespacedName, member)
	if err != nil {
		log.Error(err, "Reconcile RaftMember")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ok, err := r.reconcileStatus(member)
	if err != nil {
		log.Error(err, "Reconcile RaftMember")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *RaftMemberReconciler) reconcileStatus(member *storagev2beta2.RaftMember) (bool, error) {
	podName := types.NamespacedName{
		Namespace: member.Namespace,
		Name:      member.Spec.PodName,
	}
	pod := &corev1.Pod{}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		return false, err
	}

	state := storagev2beta2.RaftMemberReady
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			if condition.Status != corev1.ConditionTrue {
				state = storagev2beta2.RaftMemberNotReady
			}
			break
		}
	}
	log.Warnf("State for %s is %s", member.Name, state)

	if member.Status.State == nil || *member.Status.State != state {
		member.Status.State = &state
		if err := r.client.Status().Update(context.TODO(), member); err != nil {
			return false, err
		}
		return true, nil
	}

	go func() {
		err := r.startMonitoringPod(member)
		if err != nil {
			log.Error(err)
		}
	}()
	return false, nil
}

// nolint:gocyclo
func (r *RaftMemberReconciler) startMonitoringPod(member *storagev2beta2.RaftMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.streams[member.Name]
	if ok {
		return nil
	}

	pod := &corev1.Pod{}
	podName := types.NamespacedName{
		Namespace: member.Namespace,
		Name:      member.Spec.PodName,
	}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		return err
	}

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", pod.Status.PodIP, monitoringPort),
		grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := engine.NewRaftEventsClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.Subscribe(ctx, &engine.SubscribeRequest{})
	if err != nil {
		cancel()
		return err
	}

	r.streams[member.Name] = cancel
	go func() {
		defer func() {
			cancel()
			r.mu.Lock()
			delete(r.streams, member.Name)
			r.mu.Unlock()
			go func() {
				err := r.startMonitoringPod(member)
				if err != nil {
					log.Error(err)
				}
			}()
		}()

		group, err := r.getGroup(member)
		if err != nil {
			log.Error(err)
			return
		}

		partitionID := int(group.Spec.GroupID)

		memberName := types.NamespacedName{
			Namespace: member.Namespace,
			Name:      member.Name,
		}

		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}

			log.Infof("Received event %+v from %s", event, member.Name)

			switch e := event.Event.(type) {
			case *engine.RaftEvent_MemberReady:
				if int(e.MemberReady.Partition) == partitionID {
					r.recordPartitionReady(memberName, e.MemberReady, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_LeaderUpdated:
				if int(e.LeaderUpdated.Partition) == partitionID {
					r.recordLeaderUpdated(memberName, e.LeaderUpdated, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_MembershipChanged:
				if int(e.MembershipChanged.Partition) == partitionID {
					r.recordMembershipChanged(memberName, e.MembershipChanged, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SendSnapshotStarted:
				if int(e.SendSnapshotStarted.Partition) == partitionID {
					r.recordSendSnapshotStarted(memberName, e.SendSnapshotStarted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SendSnapshotCompleted:
				if int(e.SendSnapshotCompleted.Partition) == partitionID {
					r.recordSendSnapshotCompleted(memberName, e.SendSnapshotCompleted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SendSnapshotAborted:
				if int(e.SendSnapshotAborted.Partition) == partitionID {
					r.recordSendSnapshotAborted(memberName, e.SendSnapshotAborted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SnapshotReceived:
				if int(e.SnapshotReceived.Partition) == partitionID {
					r.recordSnapshotReceived(memberName, e.SnapshotReceived, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SnapshotRecovered:
				if int(e.SnapshotRecovered.Partition) == partitionID {
					r.recordSnapshotRecovered(memberName, e.SnapshotRecovered, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SnapshotCreated:
				if int(e.SnapshotCreated.Partition) == partitionID {
					r.recordSnapshotCreated(memberName, e.SnapshotCreated, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_SnapshotCompacted:
				if int(e.SnapshotCompacted.Partition) == partitionID {
					r.recordSnapshotCompacted(memberName, e.SnapshotCompacted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_LogCompacted:
				if int(e.LogCompacted.Partition) == partitionID {
					r.recordLogCompacted(memberName, e.LogCompacted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_LogdbCompacted:
				if int(e.LogdbCompacted.Partition) == partitionID {
					r.recordLogDBCompacted(memberName, e.LogdbCompacted, metav1.NewTime(event.Timestamp))
				}
			case *engine.RaftEvent_ConnectionEstablished:
				r.recordConnectionEstablished(memberName, e.ConnectionEstablished, metav1.NewTime(event.Timestamp))
			case *engine.RaftEvent_ConnectionFailed:
				r.recordConnectionFailed(memberName, e.ConnectionFailed, metav1.NewTime(event.Timestamp))
			}
		}
	}()
	return nil
}

func (r *RaftMemberReconciler) getMember(memberName types.NamespacedName) (*storagev2beta2.RaftMember, error) {
	member := &storagev2beta2.RaftMember{}
	if err := r.client.Get(context.TODO(), memberName, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (r *RaftMemberReconciler) getGroup(member *storagev2beta2.RaftMember) (*storagev2beta2.RaftGroup, error) {
	for _, ref := range member.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			groupName := types.NamespacedName{
				Namespace: member.Namespace,
				Name:      ref.Name,
			}
			group := &storagev2beta2.RaftGroup{}
			if err := r.client.Get(context.TODO(), groupName, group); err != nil {
				return nil, err
			}
			return group, nil
		}
	}
	return nil, errors.New("invalid group member")
}

func (r *RaftMemberReconciler) getPod(member *storagev2beta2.RaftMember) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	podName := types.NamespacedName{
		Namespace: member.Namespace,
		Name:      member.Spec.PodName,
	}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (r *RaftMemberReconciler) recordPartitionReady(memberName types.NamespacedName, event *engine.MemberReadyEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "MemberReady", "Member is ready to receive requests on partition %d", event.Partition)
	r.events.Eventf(member, "Normal", "Ready", "Member is ready to receive requests")

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		state := storagev2beta2.RaftMemberReady
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
			State:       &state,
			LastUpdated: &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *RaftMemberReconciler) recordLeaderUpdated(memberName types.NamespacedName, event *engine.LeaderUpdatedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}

	if member.Status.Term == nil || *member.Status.Term != event.Term {
		r.events.Eventf(member, "Normal", "TermChanged", "Term changed to %d", event.Term)
		r.events.Eventf(pod, "Normal", "GroupTermChanged", "Term for partition %d changed to %d", event.Partition, event.Term)
	}

	if event.Leader != "" && (member.Status.Leader == nil || *member.Status.Leader != event.Leader) {
		r.events.Eventf(member, "Normal", "LeaderChanged", "Leader for term %d changed to %s", event.Term, event.Leader)
		r.events.Eventf(pod, "Normal", "GroupLeaderChanged", "Leader for partition %d changed to %s for term %d", event.Partition, event.Leader, event.Term)
	}

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		role := storagev2beta2.RaftFollower
		if event.Leader == member.Spec.PodName {
			role = storagev2beta2.RaftLeader
		}
		var leader *string
		if event.Leader != "" {
			leader = &event.Leader
		}
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
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

func (r *RaftMemberReconciler) recordMembershipChanged(memberName types.NamespacedName, event *engine.MembershipChangedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupMembershipChanged", "Membership changed in group %d", event.Partition)
	r.events.Eventf(member, "Normal", "MembershipChanged", "Membership changed")
}

func (r *RaftMemberReconciler) recordSendSnapshotStarted(memberName types.NamespacedName, event *engine.SendSnapshotStartedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSendSnapshotStared", "Started sending group %d snapshot at index %d to %s", event.Partition, event.Index, event.To)
	r.events.Eventf(member, "Normal", "SendSnapshotStared", "Started sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *RaftMemberReconciler) recordSendSnapshotCompleted(memberName types.NamespacedName, event *engine.SendSnapshotCompletedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSendSnapshotCompleted", "Completed sending group %d snapshot at index %d to %s", event.Partition, event.Index, event.To)
	r.events.Eventf(member, "Normal", "SendSnapshotCompleted", "Completed sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *RaftMemberReconciler) recordSendSnapshotAborted(memberName types.NamespacedName, event *engine.SendSnapshotAbortedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Warning", "GroupSendSnapshotAborted", "Aborted sending group %d snapshot at index %d to %s", event.Partition, event.Index, event.To)
	r.events.Eventf(member, "Warning", "SendSnapshotAborted", "Aborted sending snapshot at index %d to %s", event.Index, event.To)
}

func (r *RaftMemberReconciler) recordSnapshotReceived(memberName types.NamespacedName, event *engine.SnapshotReceivedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSnapshotReceived", "Group %d received snapshot at index %d from %s", event.Partition, event.Index, event.From)
	r.events.Eventf(member, "Normal", "SnapshotReceived", "Received snapshot at index %d from %s", event.Index, event.From)

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *RaftMemberReconciler) recordSnapshotRecovered(memberName types.NamespacedName, event *engine.SnapshotRecoveredEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSnapshotRecovered", "Recovered group %d from snapshot at index %d", event.Partition, event.Index)
	r.events.Eventf(member, "Normal", "SnapshotRecovered", "Recovered from snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *RaftMemberReconciler) recordSnapshotCreated(memberName types.NamespacedName, event *engine.SnapshotCreatedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSnapshotCreated", "Created group %d snapshot at index %d", event.Partition, event.Index)
	r.events.Eventf(member, "Normal", "SnapshotCreated", "Created snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *RaftMemberReconciler) recordSnapshotCompacted(memberName types.NamespacedName, event *engine.SnapshotCompactedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupSnapshotCompacted", "Compacted group %d snapshot at index %d", event.Partition, event.Index)
	r.events.Eventf(member, "Normal", "SnapshotCompacted", "Compacted snapshot at index %d", event.Index)

	err = backoff.Retry(func() error {
		member, err := r.getMember(memberName)
		if err != nil {
			log.Error(err)
		}
		now := metav1.Now()
		return r.updateMemberStatus(member, storagev2beta2.RaftMemberStatus{
			LastSnapshotIndex: &event.Index,
			LastSnapshotTime:  &now,
			LastUpdated:       &timestamp,
		})
	}, backoff.NewExponentialBackOff())
	if err != nil {
		log.Error(err)
	}
}

func (r *RaftMemberReconciler) recordLogCompacted(memberName types.NamespacedName, event *engine.LogCompactedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupLogCompacted", "Log in group %d compacted at index %d", event.Partition, event.Index)
	r.events.Eventf(member, "Normal", "LogCompacted", "Log compacted at index %d", event.Index)
}

func (r *RaftMemberReconciler) recordLogDBCompacted(memberName types.NamespacedName, event *engine.LogDBCompactedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "GroupLogDBCompacted", "LogDB in group %d compacted at index %d", event.Partition, event.Index)
	r.events.Eventf(member, "Normal", "LogDBCompacted", "LogDB compacted at index %d", event.Index)
}

func (r *RaftMemberReconciler) recordConnectionEstablished(memberName types.NamespacedName, event *engine.ConnectionEstablishedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Normal", "ConnectionEstablished", "Connection established to %s", event.Address)
}

func (r *RaftMemberReconciler) recordConnectionFailed(memberName types.NamespacedName, event *engine.ConnectionFailedEvent, timestamp metav1.Time) {
	member, err := r.getMember(memberName)
	if err != nil {
		log.Error(err)
	}
	pod, err := r.getPod(member)
	if err != nil {
		log.Error(err)
	}
	r.events.Eventf(pod, "Warning", "ConnectionFailed", "Connection to %s failed", event.Address)
}

func (r *RaftMemberReconciler) updateMemberStatus(member *storagev2beta2.RaftMember, status storagev2beta2.RaftMemberStatus) error {
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

var _ reconcile.Reconciler = (*RaftMemberReconciler)(nil)

type mapPodMember struct {
	client client.Client
}

func (m *mapPodMember) Map(object handler.MapObject) []reconcile.Request {
	members := &storagev2beta2.RaftMemberList{}
	if err := m.client.List(context.TODO(), members, &client.ListOptions{Namespace: object.Meta.GetNamespace()}); err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, 0, len(members.Items))
	for _, member := range members.Items {
		if member.Spec.PodName == object.Meta.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: member.Namespace,
					Name:      member.Name,
				},
			})
		}
	}
	return requests
}
