///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"
	"sync"
	"testing"
	"time"
)

func TestVerifyServersOnline(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{mockPostPhaseImplementation(nil),
			mockPostPhaseImplementation(nil)}, 10, t)
	defer Shutdown(comms)

	var dlck sync.Mutex
	done := 0
	go func(d *int) {
		time.Sleep(2 * time.Second)
		dlck.Lock()
		defer dlck.Unlock()
		*d = 1
	}(&done)
	VerifyServersOnline(comms[0], topology)
	dlck.Lock()
	if done == 1 {
		t.Errorf("Could not verify servers in less than 2 seconds!")
	}
	dlck.Unlock()
}
