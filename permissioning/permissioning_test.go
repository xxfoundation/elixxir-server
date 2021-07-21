///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
)

//// Full-stack happy path test for the node registration logic
func TestRegisterNode(t *testing.T) {

	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Errorf("Failed to read cert file: %+v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Errorf("Failed to read key file: %+v", err)
	}

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)

	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7100+count)
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2100+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4100+count)
	count++
	countLock.Unlock()

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        nil,
		PrivateKey:       nil,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
		Network: internal.Perm{
			TlsCert: cert,
			Address: pAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
		DevMode:         true,
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Errorf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Errorf("Unable to create instance: %+v", err)
	}

	// Upsert test data for gateway
	instance.UpsertGatewayData("0.0.0.0:5289", "1.4")

	// Start up permissioning server
	permComms, err := startPermissioning(pAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Fetch permissioning host
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		t.Errorf("Could not get permissioning host. Was it added?")
	}

	// Register node with permissioning
	err = RegisterNode(def, instance, permHost)
	if err != nil {
		t.Errorf("Failed to register node: %+v", err)
	}

}

// Happy path: Test polling
func TestPoll(t *testing.T) {
	// Create instance
	instance, pAddr, nAddr, nodeId, cert, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	// Start up permissioning server
	permComms, err := startPermissioning(pAddr, nAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Poll the permissioning server for updates
	err = Poll(instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	// Fetch the full ndf
	receivedFullNdf, err := instance.GetConsensus().GetFullNdf().Get().Marshal()
	if err != nil {
		t.Errorf("Failed to marshall full ndf: %+v", err)
	}

	// Fetch the partial ndf
	receivedPartialNdf, err := instance.GetConsensus().GetPartialNdf().Get().Marshal()
	if err != nil {
		t.Errorf("Failed to marshall partial ndf: %+v", err)
	}

	// Take the expected partial and full ndf
	expectedFullNdf, _ := testUtil.NDF.Marshal()
	expectedPartialNdf, _ := testUtil.NDF.StripNdf().Marshal()

	if !reflect.DeepEqual(receivedFullNdf, expectedFullNdf) {
		t.Errorf("Failed to build ndf in instance!"+
			"\n\tExpected: %+v"+
			"\n\n\n\tReceived: %+v", string(expectedFullNdf), string(receivedFullNdf))
	}

	// Check the partial ndf
	if !bytes.Equal(receivedPartialNdf, expectedPartialNdf) {
		t.Errorf("Failed to poll ndf correctly."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", string(expectedPartialNdf), string(receivedPartialNdf))
	}

	if instance.GetStateMachine().Get().String() != current.WAITING.String() {
		t.Errorf("In unexpected state after polling!"+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", current.WAITING, instance.GetStateMachine().Get())
	}

}

func TestPoll_ErrState(t *testing.T) {
	// Create instance
	instance, pAddr, nAddr, nodeId, cert, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}
	instance.SetTestRecoveredError(&pb.RoundError{
		Id:     0,
		NodeId: id.NewIdFromString("", id.Node, t).Marshal(),
		Error:  "",
	}, t)
	ok, err := instance.GetStateMachine().Update(current.ERROR)
	if !ok || err != nil {
		t.Errorf("Failed to update to error state: %+v", err)
	}

	// Start up permissioning server
	permComms, err := startPermissioning(pAddr, nAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Poll the permissioning server for updates
	err = Poll(instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	// Poll the permissioning server for updates
	err = Poll(instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	if instance.GetStateMachine().Get() != current.WAITING {
		t.Error("Failed to properly update state")
	}

	if instance.GetRecoveredError() != nil {
		t.Error("Did not properly clear recovered error")
	}
}

// Happy path: Pings the mock registration server for a poll response
func TestRetrieveState(t *testing.T) {
	// Create server instance
	instance, pAddr, nAddr, nodeId, cert, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}
	defer instance.GetNetwork().Shutdown()

	// Create permissioning server
	permComms, err := startPermissioning(pAddr, nAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server")
		t.FailNow()
	}
	defer permComms.Shutdown()

	// Add retrieve permissioning host from instance
	permHost, _ := instance.GetNetwork().GetHost(&id.Permissioning)

	// Ping permissioning for a state update
	response, err := PollPermissioning(permHost, instance, instance.GetStateMachine().Get())
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	// Pull the partial and full from the ndf
	partialNdfResponse := response.PartialNDF.Ndf
	fullNdfResponse := response.FullNDF.Ndf

	// Take the expected partial and full ndf
	expectedPartialNdf, _ := testUtil.NDF.StripNdf().Marshal()
	expectedFullNdf, _ := testUtil.NDF.Marshal()

	// Check the partial ndf
	if !bytes.Equal(partialNdfResponse, expectedPartialNdf) {
		t.Errorf("Failed to poll ndf correctly."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", string(expectedPartialNdf), string(partialNdfResponse))
	}

	// Check the full ndf
	if !bytes.Equal(fullNdfResponse, expectedFullNdf) {
		t.Errorf("Failed to poll ndf correctly."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", string(expectedFullNdf), string(fullNdfResponse))
	}
}

// Happy path: Transfer from not started to precomputing, then from standby to realtime
func TestUpdateInternalState(t *testing.T) {
	numUpdates := uint64(0)

	// Create server instance
	instance, _, _, _, _, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	instance.IsFirstRun()

	// Create a topology for round info
	nodeOne := id.NewIdFromUInt(0, id.Node, t).Marshal()
	nodeTwo := id.NewIdFromUInt(1, id.Node, t).Marshal()
	nodeThree := id.NewIdFromUInt(2, id.Node, t).Marshal()
	ourTopology := [][]byte{nodeOne, nodeTwo, nodeThree}

	now := time.Now()
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PRECOMPUTING] = uint64(now.UnixNano())

	// Construct round info message
	precompRoundInfo := &pb.RoundInfo{
		ID:         1,
		UpdateID:   numUpdates,
		State:      uint32(states.PRECOMPUTING),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	err = signRoundInfo(precompRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign precomp round info: %+v", err)
	}

	// Set up the ndf's
	fullNdf, err := setupFullNdf(key)
	if err != nil {
		t.Errorf("Failed to setup full ndf: %+v", err)
	}
	stripNdf, err := setupPartialNdf(key)
	if err != nil {
		t.Errorf("Failed to setup partial ndf: %+v", err)
	}

	// ------------------- TRANSFER FROM WAITING TO PRECOMP ---------------------------------------

	// Construct permissioning poll response
	mockPollResponse := &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{precompRoundInfo},
	}

	err = UpdateNDf(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}
	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}

	// Fetch the instance's full ndf
	receivedFullNdf, err := instance.GetConsensus().GetFullNdf().Get().Marshal()
	if err != nil {
		t.Errorf("Failed to marshal internal full ndf: %+v", err)
	}

	// Check that full ndf was properly updated
	if !reflect.DeepEqual(receivedFullNdf, fullNdf.Ndf) {
		t.Errorf("Full ndf mismatch after updating internal state."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", fullNdf.Ndf, receivedFullNdf)
	}

	// Fetch the instance's partial ndf
	receivedPartialNdf, err := instance.GetConsensus().GetPartialNdf().Get().Marshal()
	if err != nil {
		t.Errorf("Failed to marshal internal full ndf: %+v", err)
	}

	// Check that partial ndf was properly updated
	if !reflect.DeepEqual(receivedPartialNdf, stripNdf.Ndf) {
		t.Errorf("Full ndf mismatch after updating internal state."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", stripNdf.Ndf, receivedPartialNdf)
	}

	// Check that the state was changed
	if instance.GetStateMachine().Get() != current.PRECOMPUTING {
		t.Errorf("Unexpected state after updating internally. "+
			"\n\tExpected state: %+v"+
			"\n\tReceived state: %+v", current.PRECOMPUTING, instance.GetStateMachine().Get())
	}

	// ----------------------- TRANSFER FROM STANDBY TO REALTIME ---------------------------------------

	ok, err := instance.GetStateMachine().Update(current.STANDBY)
	if !ok || err != nil {
		t.Errorf("Failed to transition to standby state: %+v", err)
	}

	// Create a time stamp in which to transfer stats
	// Construct round info message
	realtimeRoundInfo := &pb.RoundInfo{
		ID:       2,
		UpdateID: numUpdates,
		// Queue it into starting realtime
		State:      uint32(states.QUEUED),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	err = signRoundInfo(realtimeRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign realtime round info: %+v", err)
	}

	// Construct permissioning poll response
	mockPollResponse = &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{realtimeRoundInfo},
	}
	fmt.Println("calling update")
	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}

	// Check that the state was not changed
	if instance.GetStateMachine().Get() != current.STANDBY {
		t.Errorf("Unexpected state after updating internally. "+
			"\n\tExpected state: %+v"+
			"\n\tReceived state: %+v", current.STANDBY,
			instance.GetStateMachine().Get())
	}

	// Wait for the WaitForRealtime go routine to update the state
	time.Sleep(1 * time.Second)

	// Check that the state was not changed
	if instance.GetStateMachine().Get() != current.REALTIME {
		t.Errorf("Unexpected state after updating internally. "+
			"\n\tExpected state: %+v"+
			"\n\tReceived state: %+v", current.REALTIME,
			instance.GetStateMachine().Get())
	}

}

// Smoke tests the state transitions that contain no actual logic
func TestUpdateInternalState_Smoke(t *testing.T) {
	numUpdates := uint64(0)

	// Create server instance
	instance, _, _, _, _, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	// Create a topology for round info
	nodeOne := id.NewIdFromUInt(0, id.Node, t).Marshal()
	nodeTwo := id.NewIdFromUInt(1, id.Node, t).Marshal()
	nodeThree := id.NewIdFromUInt(2, id.Node, t).Marshal()
	ourTopology := [][]byte{nodeOne, nodeTwo, nodeThree}

	// ------------------------------- PENDING TEST ------------------------------------------------------------
	now := time.Now()
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PRECOMPUTING] = uint64(now.UnixNano())
	pendingRoundInfo := &pb.RoundInfo{
		ID:         1,
		UpdateID:   numUpdates,
		State:      uint32(states.PENDING),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	err = signRoundInfo(pendingRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign pending round info: %+v", err)
	}

	// Set up the ndf's
	fullNdf, _ := setupFullNdf(key)
	stripNdf, _ := setupPartialNdf(key)

	// Construct permissioning poll response
	mockPollResponse := &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{pendingRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}

	// ------------------------------- STANDBY TESTING ------------------------------------------------------------
	standbyRoundInfo := &pb.RoundInfo{
		ID:         2,
		UpdateID:   numUpdates,
		State:      uint32(states.STANDBY),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	err = signRoundInfo(standbyRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign standby round info: %+v", err)
	}

	// Construct permissioning poll response
	mockPollResponse = &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{standbyRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}

	// ------------------------------- COMPLETED TESTING ------------------------------------------------------------
	completedRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   numUpdates,
		State:      uint32(states.COMPLETED),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	err = signRoundInfo(completedRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign completed round info: %+v", err)
	}

	// Construct permissioning poll response
	mockPollResponse = &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{completedRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}
}

// Error path: Pass in a state that is unexpected in the round info,
// Attempt to update round in which our node is not a team-member
func TestUpdateInternalState_Error(t *testing.T) {
	// Create server instance
	instance, _, _, _, _, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	instance.IsFirstRun()

	// Create a topology for round info
	nodeOne := id.NewIdFromUInt(0, id.Node, t).Marshal()
	nodeTwo := id.NewIdFromUInt(1, id.Node, t).Marshal()
	nodeThree := id.NewIdFromUInt(2, id.Node, t).Marshal()
	ourTopology := [][]byte{nodeOne, nodeTwo, nodeThree}

	// ------------------- Enter an unexpected state -------------------------------------

	now := time.Now()
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PRECOMPUTING] = uint64(now.UnixNano())

	// Construct round info message
	NumStateRoundInfo := &pb.RoundInfo{
		ID:       1,
		UpdateID: 4,
		// Attempt to turn to a state that doesn't exist (there are only NUM_STATES - 1 states)
		State:      uint32(states.NUM_STATES),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Set the signature field of the round info
	err = signRoundInfo(NumStateRoundInfo, key)
	if err != nil {
		t.Errorf("Failed to sign NumState round info: %+v", err)
	}

	// Set up the ndf's
	fullNdf, err := setupFullNdf(key)
	if err != nil {
		t.Errorf("Failed to setup full ndf: %+v", err)
	}
	stripNdf, err := setupPartialNdf(key)
	if err != nil {
		t.Errorf("Failed to setup partial ndf: %+v", err)
	}

	// Construct permissioning poll response
	mockPollResponse := &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{NumStateRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateRounds(mockPollResponse, instance)
	if err == nil {
		t.Errorf("Expected error path. Attempted to transfer to an unknown state")
	}

}

//Full-stack happy path test for the node registration logic
func TestRegistration(t *testing.T) {

	gwConnected := make(chan struct{})
	permDone := make(chan struct{})

	// Pull certs and key
	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Errorf("Failed to load cert: +%v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Errorf("Failed to load key: +%v", err)
	}

	// Generate id's and addresses
	nodeId := id.NewIdFromUInt(0, id.Node, t)
	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7400+count)
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2400+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4400+count)
	count++
	countLock.Unlock()
	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        nil,
		PrivateKey:       nil,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		MetricLogPath:    "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
		Network: internal.Perm{
			TlsCert: cert,
			Address: pAddr,
		},
		RegistrationCode: "",
		GraphGenerator:   services.GraphGenerator{},
		ResourceMonitor:  nil,
		FullNDF:          emptyNdf,
		PartialNDF:       emptyNdf,
		DevMode:          true,
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Errorf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Errorf("Unable to create instance: %+v", err)
	}

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Boot up permissioning server
	permComms, err := startPermissioning(pAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// In go func
	go func() {
		// fixme: have gateway testing supported for a full stack test?
		//time.Sleep(1 * time.Second)
		//gAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
		//gHandler := gateway.Handler(&mockGateway{})
		//gwComms = gateway.StartGateway(nodeId.NewGateway().String(), gAddr, gHandler, cert, key)
		//_, err := gwComms.AddHost(nodeId.String(), nodeAddr, cert, false, false)
		//if err != nil {
		//	t.Errorf("Failed to add gateway host")
		//}
		//if err != nil {
		//	t.Fatalf("Gateway could not connect to node")
		//}
		gwConnected <- struct{}{}
	}()

	// Upsert test data for gateway
	instance.UpsertGatewayData("0.0.0.0:5289", "1.4")

	// Register the node in a separate thread and notify when finished
	go func() {
		// Fetch permissioning host
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		permHost, err := instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address, def.Network.TlsCert, params)
		if err != nil {
			t.Errorf("Unable to connect to registration server: %+v", err)
		}

		// Register with node
		err = RegisterNode(def, instance, permHost)
		if err != nil {
			t.Error(err)
		}
		// Blocking call: Request ndf from permissioning
		err = Poll(instance)
		if err != nil {
			t.Errorf("Failed to get ndf: %+v", err)
		}

		permDone <- struct{}{}

	}()
	// wait for gateway to connect
	<-gwConnected

	// fixme: have gateway testing supported for a full stack test?
	////poll server from gateway
	//numPolls := 0
	//for {
	//	if numPolls == 10 {
	//		t.Fatalf("Gateway could not get cert from server")
	//	}
	//	numPolls++
	//	nodeHost, _ := gwComms.GetHost(nodeId.String())
	//
	//	//emptyNdf, _ := builEmptydMockNdf().Marshal()
	//
	//	serverPoll := &pb.ServerPoll{
	//	}
	//
	//	msg, err := gwComms.SendPoll(nodeHost, serverPoll)
	//	if err != nil {
	//		t.Errorf("Error on polling signed certs: %+v", err)
	//	} else if bytes.Compare(msg.IdfPath, make([]byte, 0)) != 0 { //&& msg.Ndf.Ndf !=  {
	//		break
	//	}
	//}

	//wait for server to finish
	<-permDone
}

func TestPoll_MultipleRoundupdates(t *testing.T) {
	// Create instance
	instance, pAddr, nAddr, nodeId, cert, key, err := createServerInstance(t)
	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	// Start up permissioning server which will return multiple round updates
	permComms, err := startMultipleRoundUpdatesPermissioning(pAddr, nAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Poll the permissioning server for updates
	err = Poll(instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	// todo: check internal state for changes appropriate to permissioning response

}

// TestQueueUntilRealtime tests that the queueUntilRealtime function waits the
// specified amount of time before transitioning an instance object from QUEUED
// (or STANDBY) to REALTIME states. It also checks the case when the time
// requested to wait until is after the current time.
func TestQueueUntilRealtime(t *testing.T) {
	// Test that it happens after ~100ms
	instance, _, _, _, _, _, _ := createServerInstance(t)
	now := time.Now()
	after := now.Add(100 * time.Millisecond)
	before := now.Add(-100 * time.Millisecond)
	ok, err := instance.GetStateMachine().Update(current.PRECOMPUTING)
	if err != nil {
		t.Errorf("Failed to update to precomputing: %+v", err)
	}
	if !ok {
		t.Errorf("Did not transition to precomputing")
	}
	ok, err = instance.GetStateMachine().Update(current.STANDBY)
	if err != nil {
		t.Errorf("Failed to update to standby: %+v", err)
	}
	if !ok {
		t.Errorf("Did not transition to standby")
	}
	go queueUntilRealtime(instance, after)
	if instance.GetStateMachine().Get() == current.REALTIME {
		t.Errorf("State transitioned too quickly!\n")
	}
	time.Sleep(150 * time.Millisecond)
	if instance.GetStateMachine().Get() != current.REALTIME {
		t.Errorf("State transitioned too slowly!\n")
	}

	// Test the case where time.Now() is after the start time
	instance, _, _, _, _, _, _ = createServerInstance(t)
	ok, err = instance.GetStateMachine().Update(current.PRECOMPUTING)
	if err != nil {
		t.Errorf("Failed to update to precomputing: %+v", err)
	}
	if !ok {
		t.Errorf("Did not transition to precomputing")
	}
	ok, err = instance.GetStateMachine().Update(current.STANDBY)
	if err != nil {
		t.Errorf("Failed to update to standby: %+v", err)
	}
	if !ok {
		t.Errorf("Did not transition to standby")
	}
	go queueUntilRealtime(instance, before)
	time.Sleep(50 * time.Millisecond)
	if instance.GetStateMachine().Get() != current.REALTIME {
		t.Errorf("State transitioned too slowly!\n")
	}
}

func TestUpdateRounds_Failed(t *testing.T) {

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 17000+rand.Intn(1000))
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4000+rand.Intn(1000))
	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        nil,
		PrivateKey:       nil,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
		Network: internal.Perm{
			TlsCert: []byte(testUtil.RegCert),
			Address: pAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
		DevMode:         true,
	}

	def.PrivateKey, _ = rsa.GenerateKey(crand.Reader, 1024)

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Errorf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Errorf("Failed to create instance: %+v", err)
	}

	instance.GetRoundManager().AddRound(round.NewDummyRound(id.Round(0), uint32(4), t))

	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, "0.0.0.0", cert, params)

	now := time.Now()
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PRECOMPUTING] = uint64(now.UnixNano())

	update := &pb.RoundInfo{
		ID:       uint64(0),
		UpdateID: uint64(1),
		State:    uint32(states.FAILED),
		Topology: [][]byte{
			instance.GetID().Marshal(),
		},
		Timestamps: timestamps,
	}
	loadedKey, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		t.Errorf("Failed to load PK from pem: %+v", err)
	}
	err = signature.SignRsa(update, loadedKey)
	if err != nil {
		t.Errorf("Failed to sign update: %+v", err)
	}

	err = UpdateRounds(&pb.PermissionPollResponse{
		FullNDF:    nil,
		PartialNDF: nil,
		Updates: []*pb.RoundInfo{
			update,
		},
	}, instance)
	if err != nil {
		t.Errorf("UpdateRounds failed: %+v", err)
	}
}
