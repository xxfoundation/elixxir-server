////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"

	"testing"
	"time"
)

func TestVerifyServersOnline(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{nil, mockPostPhaseImplementation()}, 10)
	defer Shutdown(comms)

	done := 0
	go func(d *int) {
		time.Sleep(2 * time.Second)
		*d = 1
	}(&done)
	VerifyServersOnline(comms[0], topology)
	if done == 1 {
		t.Errorf("Could not verify servers in less than 2 seconds!")
	}
}
