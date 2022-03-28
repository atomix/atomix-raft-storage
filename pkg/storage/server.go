// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
	log.Infof("Starting server for partition %d", s.clusterID)
	err := s.node.StartCluster(s.members, false, s.fsm, s.config)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// Stop stops the server
func (s *Server) Stop() error {
	log.Infof("Stopping server for partition %d", s.clusterID)
	err := s.node.StopCluster(s.clusterID)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
