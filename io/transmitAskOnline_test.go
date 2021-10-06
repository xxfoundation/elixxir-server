///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

//todo-get em working and renable
/*func TestVerifyServersOnline(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{mockPostPhaseImplementation(nil),
			mockPostPhaseImplementation(nil)}, 10, t)
	defer Shutdown(comms)

	err := VerifyServersOnline(comms[0], topology,
		time.Duration(1*time.Nanosecond))
	if err == nil {
		t.Errorf("Expected timeout!")
	}
	t.Logf(err.Error())

	err = VerifyServersOnline(comms[0], topology,
		time.Duration(2*time.Second))
	if err != nil {
		t.Errorf("Unexpected error: %+v", err)
	}
}*/
