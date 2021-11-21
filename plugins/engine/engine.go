// Copyright 2021-present Open Networking Foundation.
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

package main

import (
	"bytes"
	raft "github.com/atomix/atomix-raft-storage/pkg/engine"
	"github.com/atomix/atomix-raft-storage/pkg/engine/config"
	"github.com/atomix/atomix-sdk-go/pkg/cluster"
	"github.com/atomix/atomix-sdk-go/pkg/engine"
	"github.com/atomix/atomix-sdk-go/pkg/engine/node"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/counter"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/election"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/indexedmap"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/list"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/lock"
	_map "github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/map"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/set"
	"github.com/atomix/atomix-sdk-go/pkg/engine/protocol/rsm/value"
	"github.com/atomix/atomix-sdk-go/pkg/server"
	"github.com/gogo/protobuf/jsonpb"
)

// RaftEngine is a Raft storage engine
type RaftEngine struct{}

func (e RaftEngine) NewNode(cluster cluster.Cluster, server server.Node, data []byte) (node.Node, error) {
	config := config.ProtocolConfig{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), &config); err != nil {
		return nil, err
	}
	node := rsm.NewNode(cluster, raft.NewProtocol(config))
	counter.RegisterService(node)
	election.RegisterService(node)
	indexedmap.RegisterService(node)
	list.RegisterService(node)
	lock.RegisterService(node)
	_map.RegisterService(node)
	set.RegisterService(node)
	value.RegisterService(node)
	return node, nil
}

var _ engine.Engine = RaftEngine{}

var Engine RaftEngine
