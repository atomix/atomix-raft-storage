// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"github.com/atomix/atomix-go-framework/pkg/atomix/errors"
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
func newPartition(protocol *Protocol, clusterID uint64, nodeID uint64, node *dragonboat.NodeHost, streams *streamManager) *Partition {
	return &Partition{
		protocol:  protocol,
		clusterID: clusterID,
		nodeID:    nodeID,
		node:      node,
		streams:   streams,
		listeners: make(map[int]chan<- rsm.PartitionConfig),
	}
}

// Partition is a Raft partition
type Partition struct {
	protocol   *Protocol
	clusterID  uint64
	nodeID     uint64
	node       *dragonboat.NodeHost
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

// Followers returns the current set of followers' addresses for this partition
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
	leaderID := c.protocol.getNodeID(leader)
	apiAddresses := c.protocol.getAPIAddresses()
	config.Leader = apiAddresses[leaderID]
	config.Followers = make([]string, 0, len(apiAddresses))
	for id, memberAddress := range apiAddresses {
		if id != leaderID {
			config.Followers = append(config.Followers, memberAddress)
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

// WatchConfig watches the partition configuration for changes
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
	streamID := c.streams.addStream(stream)
	defer c.streams.removeStream(streamID)
	entry := &Entry{
		Value:    input,
		StreamID: streamID,
	}
	bytes, err := proto.Marshal(entry)
	if err != nil {
		return errors.NewInvalid("failed to marshal entry", err)
	}
	ctx, cancel := context.WithTimeout(ctx, clientTimeout)
	defer cancel()
	if _, err := c.node.SyncPropose(ctx, c.node.GetNoOPSession(c.clusterID), bytes); err != nil {
		return wrapError(err)
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
		return wrapError(err)
	}
	return nil
}

// StaleQuery executes a state machine query on the partition
func (c *Partition) StaleQuery(ctx context.Context, input []byte, stream streams.WriteStream) error {
	query := queryContext{
		value:  input,
		stream: stream,
	}
	if _, err := c.node.StaleRead(c.clusterID, query); err != nil {
		return wrapError(err)
	}
	return nil
}

func wrapError(err error) error {
	switch err {
	case dragonboat.ErrClusterNotFound,
		dragonboat.ErrClusterNotBootstrapped,
		dragonboat.ErrClusterNotInitialized,
		dragonboat.ErrClusterNotReady,
		dragonboat.ErrClusterClosed:
		return errors.NewUnavailable(err.Error())
	case dragonboat.ErrSystemBusy,
		dragonboat.ErrBadKey:
		return errors.NewUnavailable(err.Error())
	case dragonboat.ErrClosed,
		dragonboat.ErrNodeRemoved:
		return errors.NewUnavailable(err.Error())
	case dragonboat.ErrInvalidSession,
		dragonboat.ErrInvalidTarget,
		dragonboat.ErrInvalidAddress,
		dragonboat.ErrInvalidOperation:
		return errors.NewInvalid(err.Error())
	case dragonboat.ErrPayloadTooBig,
		dragonboat.ErrTimeoutTooSmall:
		return errors.NewForbidden(err.Error())
	case dragonboat.ErrDeadlineNotSet,
		dragonboat.ErrInvalidDeadline:
		return errors.NewInternal(err.Error())
	case dragonboat.ErrDirNotExist:
		return errors.NewInternal(err.Error())
	case dragonboat.ErrTimeout:
		return errors.NewTimeout(err.Error())
	case dragonboat.ErrCanceled:
		return errors.NewCanceled(err.Error())
	case dragonboat.ErrRejected:
		return errors.NewForbidden(err.Error())
	default:
		return errors.NewUnknown(err.Error())
	}
}

var _ rsm.Partition = &Partition{}
