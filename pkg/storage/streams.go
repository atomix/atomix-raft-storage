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
func (r *streamManager) addStream(stream streams.WriteStream) (streamID, streams.WriteStream) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	streamID := r.nextID
	r.streams[streamID] = stream
	return streamID, stream
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
