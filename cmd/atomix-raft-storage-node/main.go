// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	protocolapi "github.com/atomix/atomix-api/go/atomix/protocol"
	"github.com/atomix/atomix-go-framework/pkg/atomix/cluster"
	"github.com/atomix/atomix-go-framework/pkg/atomix/logging"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/counter"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/election"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/indexedmap"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/list"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/lock"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/map"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/set"
	"github.com/atomix/atomix-go-framework/pkg/atomix/storage/protocol/rsm/value"
	raft "github.com/atomix/atomix-raft-storage/pkg/storage"
	"github.com/atomix/atomix-raft-storage/pkg/storage/config"
	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

const monitoringPort = 5000

func main() {
	logging.SetLevel(logging.DebugLevel)

	nodeID := os.Args[1]
	protocolConfig := parseProtocolConfig()
	raftConfig := parseRaftConfig()

	// Configure the Raft protocol
	protocol := raft.NewProtocol(raftConfig)

	ctrlCluster := cluster.NewCluster(cluster.NewNetwork(), protocolapi.ProtocolConfig{}, cluster.WithMemberID(nodeID), cluster.WithPort(monitoringPort))
	member, _ := ctrlCluster.Member()
	err := member.Serve(cluster.WithService(func(server *grpc.Server) {
		raft.RegisterRaftEventsServer(server, raft.NewEventServer(protocol))
	}))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	raftCluster := cluster.NewCluster(cluster.NewNetwork(), protocolConfig, cluster.WithMemberID(nodeID))

	// Create an Atomix node
	node := rsm.NewNode(raftCluster, protocol)

	// Register primitives on the Atomix node
	counter.RegisterService(node)
	election.RegisterService(node)
	indexedmap.RegisterService(node)
	lock.RegisterService(node)
	list.RegisterService(node)
	_map.RegisterService(node)
	set.RegisterService(node)
	value.RegisterService(node)

	// Start the node
	if err := node.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for an interrupt signal
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	// Stop the node after an interrupt
	if err := node.Stop(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseProtocolConfig() protocolapi.ProtocolConfig {
	configFile := os.Args[2]
	config := protocolapi.ProtocolConfig{}
	nodeBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(nodeBytes), &config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return config
}

func parseRaftConfig() config.ProtocolConfig {
	protocolConfigFile := os.Args[3]
	protocolConfig := config.ProtocolConfig{}
	protocolBytes, err := ioutil.ReadFile(protocolConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(protocolBytes) == 0 {
		return protocolConfig
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(protocolBytes), &protocolConfig); err != nil {
		fmt.Println(err)
	}
	return protocolConfig
}
