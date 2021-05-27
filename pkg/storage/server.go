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
	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/statemachine"
)

// newServer returns a new protocol server
func newServer(clusterID uint64, members map[uint64]string, node *dragonboat.NodeHost, config config.Config, fsm func(uint64, uint64) statemachine.IStateMachine) *Server {
	return &Server{
		clusterID: clusterID,
		members:   members,
		node:      node,
		config:    config,
		fsm:       fsm,
	}
}

// Server is a Raft server
type Server struct {
	clusterID uint64
	members   map[uint64]string
	node      *dragonboat.NodeHost
	config    config.Config
	fsm       func(uint64, uint64) statemachine.IStateMachine
}

// Start starts the server
func (s *Server) Start() error {
	return s.node.StartCluster(s.members, false, s.fsm, s.config)
}

// Stop stops the server
func (s *Server) Stop() error {
	return s.node.StopCluster(s.clusterID)
}
