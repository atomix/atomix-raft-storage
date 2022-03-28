// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	protocolapi "github.com/atomix/atomix-api/go/atomix/protocol"
	"github.com/atomix/atomix-go-framework/pkg/atomix/cluster"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/env"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/counter"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/election"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/indexedmap"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/list"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/lock"
	_map "github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/map"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/set"
	"github.com/atomix/atomix-go-framework/pkg/atomix/driver/proxy/rsm/value"
	"github.com/atomix/atomix-go-framework/pkg/atomix/logging"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	logging.SetLevel(logging.DebugLevel)

	address := os.Args[1]
	parts := strings.Split(address, ":")
	host, portNum := parts[0], parts[1]
	port, err := strconv.Atoi(portNum)
	if err != nil {
		panic(err)
	}

	provider := func(c cluster.Cluster, env env.DriverEnv) proxy.Protocol {
		p := rsm.NewProtocol(c, env)
		counter.Register(p)
		election.Register(p)
		indexedmap.Register(p)
		lock.Register(p)
		list.Register(p)
		_map.Register(p)
		set.Register(p)
		value.Register(p)
		return p
	}

	// Create a Raft driver
	d := driver.NewDriver(
		cluster.NewCluster(
			cluster.NewNetwork(),
			protocolapi.ProtocolConfig{},
			cluster.WithMemberID("raft"),
			cluster.WithHost(host),
			cluster.WithPort(port)),
		provider)

	// Start the node
	if err := d.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for an interrupt signal
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	// Stop the node after an interrupt
	if err := d.Stop(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
