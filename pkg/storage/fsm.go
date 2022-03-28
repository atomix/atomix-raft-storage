// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"github.com/atomix/atomix-go-framework/pkg/atomix/cluster"
	protocol "github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm"
	"github.com/atomix/atomix-go-framework/pkg/atomix/stream"
	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v3/statemachine"
	"io"
	"sync"
)

// newStateMachine returns a new primitive state machine
func newStateMachine(cluster cluster.Cluster, partitionID protocol.PartitionID, registry *protocol.Registry, streams *streamManager) *StateMachine {
	return &StateMachine{
		partition: partitionID,
		state:     protocol.NewStateMachine(registry),
		streams:   streams,
	}
}

// StateMachine is a Raft state machine
type StateMachine struct {
	partition protocol.PartitionID
	state     protocol.StateMachine
	streams   *streamManager
	mu        sync.Mutex
}

// Update updates the state machine state
func (s *StateMachine) Update(bytes []byte) (statemachine.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tsEntry := &Entry{}
	if err := proto.Unmarshal(bytes, tsEntry); err != nil {
		return statemachine.Result{}, err
	}

	stream := s.streams.getStream(tsEntry.StreamID)
	s.state.Command(tsEntry.Value, stream)
	return statemachine.Result{}, nil
}

// Lookup queries the state machine state
func (s *StateMachine) Lookup(value interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := value.(queryContext)
	s.state.Query(query.value, query.stream)
	return nil, nil
}

// SaveSnapshot saves a snapshot of the state machine state
func (s *StateMachine) SaveSnapshot(writer io.Writer, files statemachine.ISnapshotFileCollection, done <-chan struct{}) error {
	return s.state.Snapshot(writer)
}

// RecoverFromSnapshot recovers the state machine state from a snapshot
func (s *StateMachine) RecoverFromSnapshot(reader io.Reader, files []statemachine.SnapshotFile, done <-chan struct{}) error {
	return s.state.Restore(reader)
}

// Close closes the state machine
func (s *StateMachine) Close() error {
	return nil
}

type queryContext struct {
	value  []byte
	stream stream.WriteStream
}
