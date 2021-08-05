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
	"sort"
	"sync"
	"time"
)

const clientTimeout = 30 * time.Second

// newPartition returns a new Raft consensus partition client
func newPartition(clusterID uint64, nodeID uint64, node *dragonboat.NodeHost, members map[uint64]string, streams *streamManager) *Partition {
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
	clusterID  uint64
	nodeID     uint64
	node       *dragonboat.NodeHost
	members    map[uint64]string
	streams    *streamManager
	config     rsm.PartitionConfig
	listeners  map[int]chan<- rsm.PartitionConfig
	listenerID int
	mu         sync.RWMutex
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
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.Leader
}

func (c *Partition) Followers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.Followers
}

func (c *Partition) updateConfig(leader string) {
	log.Infof("Updating leader for partition %d: %s", c.clusterID, leader)
	c.mu.Lock()
	defer c.mu.Unlock()
	config := rsm.PartitionConfig{}
	config.Leader = leader
	config.Followers = make([]string, 0, len(c.members))
	for _, member := range c.members {
		if member != leader {
			config.Followers = append(config.Followers, member)
		}
	}
	sort.Slice(config.Followers, func(i, j int) bool {
		return config.Followers[i] < config.Followers[j]
	})

	c.config = config
	for _, listener := range c.listeners {
		listener <- config
	}
}

func (c *Partition) WatchConfig(ctx context.Context, ch chan<- rsm.PartitionConfig) error {
	c.listenerID++
	id := c.listenerID
	c.mu.Lock()
	c.listeners[id] = ch
	c.mu.Unlock()
	go func() {
		c.mu.RLock()
		ch <- c.config
		c.mu.RUnlock()
		<-ctx.Done()
		c.mu.Lock()
		close(ch)
		delete(c.listeners, id)
		c.mu.Unlock()
	}()
	return nil
}

// SyncCommand executes a state machine command on the partition
func (c *Partition) SyncCommand(ctx context.Context, input []byte, stream streams.WriteStream) error {
	streamID, stream := c.streams.addStream(stream)
	defer c.streams.removeStream(streamID)
	entry := &Entry{
		Value:    input,
		StreamID: streamID,
	}
	bytes, err := proto.Marshal(entry)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.SyncPropose(ctx, c.node.GetNoOPSession(c.clusterID), bytes); err != nil {
		return err
	}
	return nil
}

// SyncQuery executes a state machine query on the partition
func (c *Partition) SyncQuery(ctx context.Context, input []byte, stream streams.WriteStream) error {
	query := queryContext{
		value:  input,
		stream: stream,
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.SyncRead(ctx, c.clusterID, query); err != nil {
		return err
	}
	return nil
}

// StaleQuery executes a state machine query on the partition
func (c *Partition) StaleQuery(ctx context.Context, input []byte, stream streams.WriteStream) error {
	query := queryContext{
		value:  input,
		stream: stream,
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.StaleRead(c.clusterID, query); err != nil {
		return err
	}
	return nil
}

var _ rsm.Partition = &Partition{}
