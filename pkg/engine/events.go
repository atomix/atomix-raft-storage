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

package engine

import (
	"context"
	"github.com/lni/dragonboat/v3/raftio"
	"sync"
	"time"
)

// NewEventServer creates a new EventServer instance
func NewEventServer(protocol *Protocol) *EventServer {
	return &EventServer{
		protocol: protocol,
	}
}

// EventServer is a server for monitoring Raft cluster events
type EventServer struct {
	protocol *Protocol
}

// Subscribe subscribes to receive events from a partition
func (e *EventServer) Subscribe(request *SubscribeRequest, stream RaftEvents_SubscribeServer) error {
	ch := make(chan RaftEvent)
	e.protocol.watch(stream.Context(), ch)
	for event := range ch {
		err := stream.Send(&event)
		if err != nil {
			return err
		}
	}
	return nil
}

type raftEventListener struct {
	protocol   *Protocol
	listeners  map[int]chan<- RaftEvent
	listenerID int
	mu         sync.RWMutex
}

func (e *raftEventListener) listen(ctx context.Context, ch chan<- RaftEvent) {
	e.listenerID++
	id := e.listenerID
	e.mu.Lock()
	e.listeners[id] = ch
	e.mu.Unlock()

	go func() {
		<-ctx.Done()
		e.mu.Lock()
		close(ch)
		delete(e.listeners, id)
		e.mu.Unlock()
	}()
}

func (e *raftEventListener) publish(event RaftEvent) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	log.Infof("Publishing RaftEvent %s", event)
	for _, listener := range e.listeners {
		listener <- event
	}
}

func (e *raftEventListener) LeaderUpdated(info raftio.LeaderInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_LeaderUpdated{
			LeaderUpdated: &LeaderUpdatedEvent{
				LeaderEvent: LeaderEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Term:   info.Term,
					Leader: e.protocol.getMemberID(info.LeaderID),
				},
			},
		},
	})
}

func (e *raftEventListener) NodeHostShuttingDown() {

}

func (e *raftEventListener) NodeUnloaded(info raftio.NodeInfo) {

}

func (e *raftEventListener) NodeReady(info raftio.NodeInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_MemberReady{
			MemberReady: &MemberReadyEvent{
				PartitionEvent: PartitionEvent{
					Partition: uint32(info.ClusterID),
				},
			},
		},
	})
}

func (e *raftEventListener) MembershipChanged(info raftio.NodeInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_MembershipChanged{
			MembershipChanged: &MembershipChangedEvent{
				PartitionEvent: PartitionEvent{
					Partition: uint32(info.ClusterID),
				},
			},
		},
	})
}

func (e *raftEventListener) ConnectionEstablished(info raftio.ConnectionInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_ConnectionEstablished{
			ConnectionEstablished: &ConnectionEstablishedEvent{
				ConnectionEvent: ConnectionEvent{
					Address:  info.Address,
					Snapshot: info.SnapshotConnection,
				},
			},
		},
	})
}

func (e *raftEventListener) ConnectionFailed(info raftio.ConnectionInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_ConnectionFailed{
			ConnectionFailed: &ConnectionFailedEvent{
				ConnectionEvent: ConnectionEvent{
					Address:  info.Address,
					Snapshot: info.SnapshotConnection,
				},
			},
		},
	})
}

func (e *raftEventListener) SendSnapshotStarted(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SendSnapshotStarted{
			SendSnapshotStarted: &SendSnapshotStartedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
				To: e.protocol.getMemberID(info.NodeID),
			},
		},
	})
}

func (e *raftEventListener) SendSnapshotCompleted(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SendSnapshotCompleted{
			SendSnapshotCompleted: &SendSnapshotCompletedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
				To: e.protocol.getMemberID(info.NodeID),
			},
		},
	})
}

func (e *raftEventListener) SendSnapshotAborted(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SendSnapshotAborted{
			SendSnapshotAborted: &SendSnapshotAbortedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
				To: e.protocol.getMemberID(info.NodeID),
			},
		},
	})
}

func (e *raftEventListener) SnapshotReceived(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SnapshotReceived{
			SnapshotReceived: &SnapshotReceivedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
				From: e.protocol.getMemberID(info.From),
			},
		},
	})
}

func (e *raftEventListener) SnapshotRecovered(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SnapshotRecovered{
			SnapshotRecovered: &SnapshotRecoveredEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
			},
		},
	})
}

func (e *raftEventListener) SnapshotCreated(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SnapshotCreated{
			SnapshotCreated: &SnapshotCreatedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
			},
		},
	})
}

func (e *raftEventListener) SnapshotCompacted(info raftio.SnapshotInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_SnapshotCompacted{
			SnapshotCompacted: &SnapshotCompactedEvent{
				SnapshotEvent: SnapshotEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
			},
		},
	})
}

func (e *raftEventListener) LogCompacted(info raftio.EntryInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_LogCompacted{
			LogCompacted: &LogCompactedEvent{
				LogEvent: LogEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
			},
		},
	})
}

func (e *raftEventListener) LogDBCompacted(info raftio.EntryInfo) {
	e.publish(RaftEvent{
		Timestamp: time.Now(),
		Event: &RaftEvent_LogdbCompacted{
			LogdbCompacted: &LogDBCompactedEvent{
				LogEvent: LogEvent{
					PartitionEvent: PartitionEvent{
						Partition: uint32(info.ClusterID),
					},
					Index: info.Index,
				},
			},
		},
	})
}
