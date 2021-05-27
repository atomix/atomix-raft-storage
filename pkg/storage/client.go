// Copyright 2019-present Open Networking Foundation.
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

package storage

import (
	"context"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm"
	streams "github.com/atomix/atomix-go-framework/pkg/atomix/stream"
	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v3"
	"time"
)

const clientTimeout = 30 * time.Second

// newClient returns a new Raft consensus protocol client
func newClient(clusterID uint64, nodeID uint64, node *dragonboat.NodeHost, members map[uint64]string, streams *streamManager) *Partition {
	return &Partition{
		clusterID: clusterID,
		nodeID:    nodeID,
		node:      node,
		members:   members,
		streams:   streams,
	}
}

// Partition is a Raft partition
type Partition struct {
	clusterID uint64
	nodeID    uint64
	node      *dragonboat.NodeHost
	members   map[uint64]string
	streams   *streamManager
}

// MustLeader returns whether the Raft partition requires a leader
func (c *Partition) MustLeader() bool {
	return true
}

// IsLeader returns whether the local node is the leader for this partition
func (c *Partition) IsLeader() bool {
	leader, ok, err := c.node.GetLeaderID(c.clusterID)
	if !ok || err != nil {
		return false
	}
	return leader == c.nodeID
}

// Leader returns the leader address for this partition
func (c *Partition) Leader() string {
	leader, ok, err := c.node.GetLeaderID(c.clusterID)
	if !ok || err != nil {
		return ""
	}
	return c.members[leader]
}

// ExecuteCommand executes a state machine command on the partition
func (c *Partition) ExecuteCommand(ctx context.Context, input []byte, stream streams.WriteStream) error {
	streamID, stream := c.streams.addStream(stream)
	entry := &Entry{
		Value:     input,
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
	bytes, err := proto.Marshal(entry)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.SyncPropose(ctx, c.node.GetNoOPSession(c.clusterID), bytes); err != nil {
		stream.Close()
		return err
	}
	return nil
}

// ExecuteQuery executes a state machine query on the partition
func (c *Partition) ExecuteQuery(ctx context.Context, input []byte, stream streams.WriteStream) error {
	query := queryContext{
		value:  input,
		stream: stream,
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.SyncRead(ctx, c.clusterID, query); err != nil {
		stream.Close()
		return err
	}
	return nil
}

var _ rsm.Partition = &Partition{}
