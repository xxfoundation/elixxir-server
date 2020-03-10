////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/node/receivers"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------

func builEmptydMockNdf() *ndf.NetworkDefinition {

	cmixGroup := ndf.Group{
		Prime:      "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		SmallPrime: "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		Generator:  "02",
	}

	e2eGroup := ndf.Group{
		Prime:      "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		SmallPrime: "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		Generator:  "02",
	}

	ourMockNdf := &ndf.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{},
		Gateways:  []ndf.Gateway{},
		E2E:       e2eGroup,
		CMIX:      cmixGroup,
		UDB:       ndf.UDB{},
	}

	return ourMockNdf
}

func buildMockNdf(nodeId *id.Node, nodeAddress, gwAddress string, cert, key []byte) {
	node := ndf.Node{
		ID:             nodeId.Bytes(),
		TlsCertificate: string(cert),
		Address:        nodeAddress,
	}
	gw := ndf.Gateway{
		Address:        gwAddress,
		TlsCertificate: string(cert),
	}
	mockGroup := ndf.Group{
		Prime:      "25",
		SmallPrime: "42",
		Generator:  "2",
	}
	testNdf = &ndf.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{node},
		Gateways:  []ndf.Gateway{gw},
		E2E:       mockGroup,
		CMIX:      mockGroup,
		UDB:       ndf.UDB{},
	}
}

//// Full-stack happy path test for the node registration logic
//func TestRegisterNode(t *testing.T) {
//
//	gwConnected := make(chan struct{})
//	permDone := make(chan struct{})
//
//	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
//	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())
//
//	nodeId = id.NewNodeFromUInt(uint64(0), t)
//	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000))
//
//	// Initialize permissioning server
//	pAddr := fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
//	pHandler := registration.Handler(&mockPermission{})
//	permComms = registration.StartRegistrationServer("ptest", pAddr, pHandler, cert, key)
//	_, err := permComms.AddHost(nodeId.String(), pAddr, cert, false, false)
//	if err != nil {
//		t.Fatalf("Permissioning could not connect to node")
//	}
//
//	gAddr := fmt.Sprintf("0.0.0.0:%d", 4000+rand.Intn(1000))
//	gHandler := gateway.Handler(&mockGateway{})
//	gwComms = gateway.StartGateway("gtest", gAddr, gHandler, cert, key)
//	buildMockNdf(nodeId, nodeAddr, gAddr, cert, key)
//	go func() {
//		time.Sleep(1 * time.Second)
//		gwComms.AddHost(nodeId.String(), gAddr, cert, false, false)
//		if err != nil {
//			t.Fatalf("Gateway could not connect to node")
//		}
//		gwConnected <- struct{}{}
//	}()
//
//	// Initialize definition
//	def := &server.Definition{
//		Flags:         server.Flags{},
//		ID:            nodeId,
//		PublicKey:     nil,
//		PrivateKey:    nil,
//		TlsCert:       cert,
//		TlsKey:        key,
//		Address:       nodeAddr,
//		LogPath:       "",
//		MetricLogPath: "",
//		Gateway: server.GW{
//			ID: nodeId.NewGateway(),
//			Address: gAddr,
//			TlsCert: cert,
//		},
//		UserRegistry:    nil,
//		GraphGenerator:  services.GraphGenerator{},
//		ResourceMonitor: nil,
//		Permissioning: server.Perm{
//			TlsCert:          cert,
//			RegistrationCode: "",
//			Address:          pAddr,
//		},
//		FullNDF:testNdf,
//		PartialNDF:testNdf,
//	}
//
//	sm := state.NewMachine(dummyStates)
//	impl := func(i *server.Instance) *node.Implementation {
//		return receivers.NewImplementation(i)
//	}
//
//	instance, err := server.CreateServerInstance(def, impl, sm, true)
//
//	// Register the node in a separate thread and notify when finished
//	go func() {
//
//		//network := node.StartNode("nodeid", def.Address, impl(instance), def.TlsCert, def.TlsKey)
//		permHost, err := instance.GetNetwork().AddHost(id.PERMISSIONING, def.Permissioning.Address, def.Permissioning.TlsCert, true, false)
//		if err != nil {
//			t.Errorf("Unable to connect to registration server: %+v", err)
//		}
//
//		err = RegisterNode(def, instance.GetNetwork(), permHost)
//		if err != nil {
//			t.Error(err)
//		}
//		// Blocking call: Request ndf from permissioning
//		err = Poll(permHost, instance)
//		if err != nil {
//			t.Errorf("Failed to get ndf: %+v", err)
//		}
//		// Parse the Nd
//		serverCert, gwCert, err := InstallNdf(def, instance.GetConsensus().GetFullNdf().Get())
//		if err != nil {
//			t.Errorf("Failed to install ndf: %+v", err)
//		}
//		def.TlsCert = []byte(serverCert)
//		def.Gateway.TlsCert = []byte(gwCert)
//		permDone <- struct{}{}
//	}()
//	// wait for gateway to connect
//	<-gwConnected
//
//	//poll server from gateway
//	numPolls := 0
//	for {
//		if numPolls == 10 {
//			t.Fatalf("Gateway could not get cert from server")
//		}
//		numPolls++
//		nodeHost, _ := gwComms.GetHost(nodeId.String())
//
//		serverPoll := &pb.ServerPoll{
//			Full:                 nil,
//			Partial:              nil,
//			LastUpdate:           0,
//			Error:                "",
//		}
//
//		msg, err := gwComms.SendPoll(nodeHost, serverPoll)
//		if err != nil {
//			t.Errorf("Error on polling signed certs: %+v", err)
//		} else if bytes.Compare(msg.Id, make([]byte, 0)) != 0 { //&& msg.Ndf.Ndf !=  {
//			break
//		}
//	}
//
//	//wait for server to finish
//	<-permDone
//
////=	if bytes.Compare(n[0].ID.Bytes(), nodeId.Bytes()) != 0 {
////		t.Errorf("Received network topology with incorrect node ID!")
////	}
////	if n[0].Address != nodeAddr && strings.Replace(n[0].Address, "127.0.0.1",
////		"0.0.0.0", -1) != nodeAddr {
////		t.Errorf("Received network topology with incorrect node address!")
////	}
////	if n[0].TlsCert == nil {
////		t.Errorf("Received network topology with incorrect node TLS cert!")
////	}
//}

func TestPoll(t *testing.T) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000))

	// Initialize permissioning server
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer("ptest", pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId.String(), pAddr, cert, false, false)
	if err != nil {
		t.Fatalf("Permissioning could not connect to node")
	}

	gAddr := fmt.Sprintf("0.0.0.0:%d", 4000+rand.Intn(1000))
	buildMockNdf(nodeId, nodeAddr, gAddr, cert, key)

	emptyNdf := builEmptydMockNdf()
	fmt.Print("our testndf: ", testNdf)
	// Initialize definition
	def := &server.Definition{
		Flags:           server.Flags{},
		ID:              nodeId,
		PublicKey:       nil,
		PrivateKey:      nil,
		TlsCert:         cert,
		TlsKey:          key,
		Address:         nodeAddr,
		LogPath:         "",
		MetricLogPath:   "",
		UserRegistry:    nil,
		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		Permissioning: server.Perm{
			TlsCert:          cert,
			RegistrationCode: "",
			Address:          pAddr,
		},
		FullNDF:    emptyNdf,
		PartialNDF: emptyNdf,
	}

	sm := state.NewMachine(dummyStates)
	impl := func(i *server.Instance) *node.Implementation {
		return receivers.NewImplementation(i)
	}

	instance, err := server.CreateServerInstance(def, impl, sm, true)
	permHost, err := instance.GetNetwork().AddHost(id.PERMISSIONING, def.Permissioning.Address,
		def.Permissioning.TlsCert, true, false)

	err = Poll(permHost, instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	if !reflect.DeepEqual(instance.GetConsensus().GetFullNdf().Get(), testNdf) {
		t.Errorf("Failed to build ndf in instance!"+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", testNdf, instance.GetConsensus().GetFullNdf().Get())
	}

}

// Happy path: Pings the mock registration server for a poll response
func TestRetrieveState(t *testing.T) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000))

	// Initialize permissioning server
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer("ptest", pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId.String(), pAddr, cert, false, false)
	if err != nil {
		t.Fatalf("Permissioning could not connect to node")
	}

	// Build the node
	emptyNdf := builEmptydMockNdf()
	// Initialize definition
	def := &server.Definition{
		Flags:           server.Flags{},
		ID:              nodeId,
		PublicKey:       nil,
		PrivateKey:      nil,
		TlsCert:         cert,
		TlsKey:          key,
		Address:         nodeAddr,
		LogPath:         "",
		MetricLogPath:   "",
		UserRegistry:    nil,
		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		Permissioning: server.Perm{
			TlsCert:          cert,
			RegistrationCode: "",
			Address:          pAddr,
		},
		FullNDF:    emptyNdf,
		PartialNDF: emptyNdf,
	}

	// Create instance
	sm := state.NewMachine(dummyStates)
	impl := func(i *server.Instance) *node.Implementation {
		return receivers.NewImplementation(i)
	}

	instance, err := server.CreateServerInstance(def, impl, sm, true)
	if err != nil {
		t.Errorf("Unable to create instance: %+v", err)
	}
	// Add permissioning as a host
	permHost, err := instance.GetNetwork().AddHost(id.PERMISSIONING, def.Permissioning.Address,
		def.Permissioning.TlsCert, false, false)
	if err != nil {
		t.Errorf("Failed to add permissioning as host in instance: %+v", err)
	}

	// Ping permissioning for a state update
	responnse, err := RetrieveState(permHost, instance)
	if err != nil {
		t.Errorf("Failed to poll for ndf: %+v", err)
	}

	// Pull the partial and full from the ndf
	partialNdfResponse := responnse.PartialNDF.Ndf
	fullNdfResponse := responnse.FullNDF.Ndf

	// Take the expected partial and full ndf
	expectedPartialNdf, _ := testUtil.NDF.StripNdf().Marshal()
	expectedFullNdf, _ := testUtil.NDF.Marshal()

	if !bytes.Equal(partialNdfResponse, expectedPartialNdf) {
		t.Errorf("Failed to poll ndf correctly."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", string(expectedPartialNdf), string(partialNdfResponse))
	}

	if !bytes.Equal(fullNdfResponse, expectedFullNdf) {
		t.Errorf("Failed to poll ndf correctly."+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", string(expectedFullNdf), string(fullNdfResponse))
	}
}

// Happy path: Transfer from not started to precomputing, then from standby to realtime
func TestUpdateInternalState_PrecompTransition(t *testing.T) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000))

	// Build the node
	emptyNdf := builEmptydMockNdf()
	// Initialize definition
	def := &server.Definition{
		Flags:         server.Flags{},
		ID:            nodeId,
		PublicKey:     nil,
		PrivateKey:    nil,
		TlsCert:       cert,
		TlsKey:        key,
		Address:       nodeAddr,
		LogPath:       "",
		MetricLogPath: "",
		UserRegistry:  nil,
		Permissioning: server.Perm{
			TlsCert:          []byte(testUtil.OurCert),
			RegistrationCode: "",
		},

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Errorf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *server.Instance) *node.Implementation {
		return receivers.NewImplementation(i)
	}

	// Generate instance
	instance, err := server.CreateServerInstance(def, impl, sm, true)
	if err != nil {
		t.Errorf("Unable to create instance: %+v", err)
	}

	// Add permissioning as a host
	_, err = instance.GetNetwork().AddHost(id.PERMISSIONING, def.Permissioning.Address,
		def.Permissioning.TlsCert, false, false)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Create a topology for round info
	nodeOne := id.NewNodeFromUInt(uint64(0), t).String()
	nodeTwo := id.NewNodeFromUInt(uint64(1), t).String()
	nodeThree := id.NewNodeFromUInt(uint64(2), t).String()
	ourTopology := []string{nodeOne, nodeTwo, nodeThree}

	// Construct round info message
	precompRoundInfo := &pb.RoundInfo{
		ID:       0,
		UpdateID: 4,
		State:    uint32(states.PRECOMPUTING),
		Topology: ourTopology,
	}

	// Set the signature field of the round info
	signRoundInfo(t, precompRoundInfo)

	// Set up the ndf's
	fullNdf := setupFullNdf(t)
	stripNdf := setupPartialNdf(t)

	// Construct permissioning poll response
	mockPollResponse := &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{precompRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateInternalState(mockPollResponse, instance)
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

	// ------------------- TRANSFER FROM PRECOMPUTING TO REALTIME

	ok, err = instance.GetStateMachine().Update(current.STANDBY)
	if !ok || err != nil {
		t.Errorf("Failed to transition to standby state: %+v", err)
	}

	// Create a time stamp in which to transfer stats
	ourTime := time.Now().Add(50 * time.Millisecond).UnixNano()
	timestamps := make([]uint64, states.FAILED)
	timestamps[states.REALTIME] = uint64(ourTime)

	// Construct round info message
	realtimeRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   5,
		State:      uint32(states.REALTIME),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Set the signature field of the round info
	signRoundInfo(t, realtimeRoundInfo)

	// Construct permissioning poll response
	mockPollResponse = &pb.PermissionPollResponse{
		FullNDF:    fullNdf,
		PartialNDF: stripNdf,
		Updates:    []*pb.RoundInfo{realtimeRoundInfo},
	}

	// Update internal state with mock response
	err = UpdateInternalState(mockPollResponse, instance)
	if err != nil {
		t.Errorf("Failed to update internal state: %+v", err)
	}

	// Wait for the WaitForRealtime go thread to update the state
	time.Sleep(50 * time.Millisecond)

	// Check that the state was changed
	if instance.GetStateMachine().Get() != current.REALTIME {
		t.Errorf("Unexpected state after updating internally. "+
			"\n\tExpected state: %+v"+
			"\n\tReceived state: %+v", current.REALTIME, instance.GetStateMachine().Get())
	}

	//states.NUM_STATES
	//// Create a round info that
	//realtimeRoundInfo := &pb.RoundInfo{
	//	ID:                   9,
	//	UpdateID:             4,
	//	State:                uint32(states.REALTIME),
	//	Topology:             ourTopology,
	//}
	//
	//// Create a
	//pendingRoundInfo := &pb.RoundInfo{
	//	ID:                   9,
	//	UpdateID:             4,
	//	State:                uint32(states.PENDING),
	//	Topology:             ourTopology,
	//}
	//
	//// Create a bad round info that goes past the number of states
	//badRoundInfo := &pb.RoundInfo{
	//	ID:                   9,
	//	UpdateID:             4,
	//	State:                uint32(states.NUM_STATES),
	//	Topology:             ourTopology,
	//}
	//
	//roundInfo := []*pb.RoundInfo{precompRoundInfo, realtimeRoundInfo, pendingRoundInfo,badRoundInfo }

}

//func TestRetrieveState(t *testing.T) {
//	impl := func(*server.Instance) *node.Implementation {
//		return node.NewImplementation()
//	}
//	def := mockServerDef(t)
//	sm := state.NewMachine(dummyStates)
//	instance, _ := server.CreateServerInstance(def, impl, sm, false)
//	network := node.StartNode("nodeid", def.Address, impl(instance), def.TlsCert, def.TlsKey)
//
//	permHost, err := network.AddHost(id.PERMISSIONING, def.Permissioning.Address, def.Permissioning.TlsCert, true, false)
//	if err != nil {
//		t.Errorf("Couldn't add permHost: %+v", err)
//	}
//
//	RetrieveState(permHost, instance)
//}

func signRoundInfo(t *testing.T, ri *pb.RoundInfo) {
	pk, err := tls.LoadRSAPrivateKey(testUtil.OurPrivateKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)

}

func setupFullNdf(t *testing.T) *mixmessages.NDF {
	pk, err := tls.LoadRSAPrivateKey(testUtil.OurPrivateKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}
	tmpNdf, _, _ := ndf.DecodeNDF(testUtil.ExampleJSON)
	f.Ndf, err = tmpNdf.Marshal()
	if err != nil {
		t.Errorf("Failed to marshal ndf: %+v", err)
	}

	if err != nil {
		t.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f
}

func setupPartialNdf(t *testing.T) *mixmessages.NDF {
	pk, err := tls.LoadRSAPrivateKey(testUtil.OurPrivateKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}

	stipped, err := testUtil.NDF.StripNdf().Marshal()
	f.Ndf = stipped

	if err != nil {
		t.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f
}
