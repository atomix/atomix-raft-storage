package storage

import (
	"context"
	"github.com/lni/dragonboat/v3/raftio"
	"sync"
	"time"
)

func NewEventServer(protocol *Protocol) *EventServer {
	ch := make(chan RaftEvent)
	protocol.watch(context.Background(), ch)
	return &EventServer{
		ch: ch,
	}
}

type EventServer struct {
	ch chan RaftEvent
}

func (e *EventServer) Subscribe(request *SubscribeRequest, stream RaftEvents_SubscribeServer) error {
	for event := range e.ch {
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
						Partition: info.ClusterID,
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
					Partition: info.ClusterID,
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
					Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
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
						Partition: info.ClusterID,
					},
					Index: info.Index,
				},
			},
		},
	})
}
