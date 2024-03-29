// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	core "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	primitives "github.com/atomix/atomix-controller/pkg/apis/primitives/v2beta1"
	logutil "github.com/atomix/atomix-controller/pkg/controller/util/log"
	"github.com/atomix/atomix-go-framework/pkg/atomix/logging"
	storagev2beta2 "github.com/atomix/atomix-raft-storage/pkg/controller/storage/v2beta2"

	"os"
	"runtime"

	"github.com/atomix/atomix-controller/pkg/controller/util/leader"
	"github.com/atomix/atomix-controller/pkg/controller/util/ready"
	"github.com/atomix/atomix-raft-storage/pkg/apis"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logging.GetLogger("main")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	logging.SetLevel(logging.InfoLevel)
	logf.SetLogger(logutil.NewControllerLogger("atomix", "controller", "raft"))

	var namespace string
	if len(os.Args) > 1 {
		namespace = os.Args[1]
	}

	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	_ = leader.Become(context.TODO())

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	defer func() {
		_ = r.Unset()
	}()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup Scheme for primitive resources
	if err := primitives.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup Scheme for storage resources
	if err := core.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add the storage/v2beta2 controllers
	if err := storagev2beta2.AddControllers(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "controller exited non-zero")
		os.Exit(1)
	}
}
