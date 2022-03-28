// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v2beta2

import (
	"github.com/atomix/atomix-go-framework/pkg/atomix/logging"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logging.GetLogger("atomix", "controller", "raft")

// AddControllers creates a new Partition ManagementGroup and adds it to the Manager. The Manager will set fields on the ManagementGroup
// and Start it when the Manager is Started.
func AddControllers(mgr manager.Manager) error {
	if err := addMultiRaftProtocolController(mgr); err != nil {
		return err
	}
	if err := addMultiRaftClusterController(mgr); err != nil {
		return err
	}
	if err := addRaftGroupController(mgr); err != nil {
		return err
	}
	if err := addRaftMemberController(mgr); err != nil {
		return err
	}
	return nil
}
