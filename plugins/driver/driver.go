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
	raft "github.com/atomix/atomix-raft-storage/pkg/driver"
	"github.com/atomix/atomix-sdk-go/pkg/cluster"
	"github.com/atomix/atomix-sdk-go/pkg/driver"
	"github.com/atomix/atomix-sdk-go/pkg/driver/agent"
	"github.com/atomix/atomix-sdk-go/pkg/driver/protocol/generic/counter"
	_map "github.com/atomix/atomix-sdk-go/pkg/driver/protocol/generic/map"
)

// RaftDriver is a multi-raft storage driver
type RaftDriver struct{}

func (d RaftDriver) NewAgent(cluster cluster.Cluster) agent.Agent {
	agent := raft.NewAgent(cluster)
	counter.RegisterProxy(agent)
	_map.RegisterProxy(agent)
	return agent
}

var _ driver.Driver = RaftDriver{}

var Driver RaftDriver
