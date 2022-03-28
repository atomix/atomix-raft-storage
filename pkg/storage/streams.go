// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	streams "github.com/atomix/atomix-go-framework/pkg/atomix/stream"
	"sync"
)

type streamID uint64

// newStreamManager returns a new stream manager
func newStreamManager() *streamManager {
	return &streamManager{
		streams: make(map[streamID]streams.WriteStream),
	}
}

// streamManager is a manager of client streams
type streamManager struct {
	streams map[streamID]streams.WriteStream
	nextID  streamID
	mu      sync.RWMutex
}

// addStream adds a new stream
func (r *streamManager) addStream(stream streams.WriteStream) streamID {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	streamID := r.nextID
	r.streams[streamID] = stream
	return streamID
}

// removeStream removes a stream by ID
func (r *streamManager) removeStream(streamID streamID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, streamID)
}

// getStream gets a stream by ID
func (r *streamManager) getStream(streamID streamID) streams.WriteStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stream, ok := r.streams[streamID]
	if ok {
		return stream
	}
	return streams.NewNilStream()
}
