package main

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	nodeComms "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var errWg = sync.WaitGroup{}

func Test_MultiInstance_N3_B8(t *testing.T) {
	MultiInstanceTest(3, 32, false, false, t)
}

func Test_MultiInstance_N3_B32_GPU(t *testing.T) {
	MultiInstanceTest(3, 32, true, false, t)
}

func Test_MultiInstance_PhaseErr(t *testing.T) {
	MultiInstanceTest(3, 32, false, true, t)
}

func MultiInstanceTest(numNodes, batchsize int, useGPU, errorPhase bool, t *testing.T) {
	if errorPhase {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
	}
	jww.SetStdoutThreshold(jww.LevelDebug)

	if numNodes < 3 {
		t.Errorf("Multi Instance Test must have a minnimum of 3 nodes,"+
			" Recieved %v", numNodes)
	}

	grp := makeMultiInstanceGroup()

	//get parameters
	portOffset := int(rand.Uint32() % 2000)
	defsLst := makeMultiInstanceParams(numNodes, 20000+portOffset, useGPU)

	//make user for sending messages
	userID := id.NewUserFromUint(42, t)
	var baseKeys []*cyclic.Int
	for i := 0; i < numNodes; i++ {
		baseKey := grp.NewIntFromUInt(uint64(1000 + 5*i))
		baseKeys = append(baseKeys, baseKey)
	}

	//build the registries for every node
	for i := 0; i < numNodes; i++ {
		var registry globals.UserRegistry
		registry = &globals.UserMap{}
		user := globals.User{
			ID:           userID,
			BaseKey:      baseKeys[i],
			IsRegistered: true,
		}
		registry.UpsertUser(&user)
		defsLst[i].UserRegistry = registry
	}

	// build the instances
	var instances []*internal.Instance

	t.Logf("Building instances for %v nodes", numNodes)

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})

	for i := 0; i < numNodes; i++ {
		var instance *internal.Instance

		// Add handler for instance
		impl := func(i *internal.Instance) *nodeComms.Implementation {
			return io.NewImplementation(i)
		}

		// Construct the state machine
		var testStates [current.NUM_STATES]state.Change
		// Create not started
		testStates[current.NOT_STARTED] = func(from current.Activity) error {
			curActivity, err := instance.GetStateMachine().WaitFor(1*time.Second, current.NOT_STARTED)
			if curActivity != current.NOT_STARTED || err != nil {
				t.Errorf("Server never transitioned to %v state: %+v", current.NOT_STARTED, err)
			}

			jww.DEBUG.Printf("Updating to WAITING")
			ok, err := instance.GetStateMachine().Update(current.WAITING)
			if !ok || err != nil {
				t.Errorf("Unable to transition to %v state: %+v", current.WAITING, err)
			}

			return nil
		}
		// Create waiting
		testStates[current.WAITING] = func(from current.Activity) error { return nil }
		// Create precomputing
		testStates[current.PRECOMPUTING] = func(from current.Activity) error {
			return node.Precomputing(instance, 5*time.Second)
		}
		// Create standby
		testStates[current.STANDBY] = func(from current.Activity) error { return nil }
		// Create realtime
		testStates[current.REALTIME] = func(from current.Activity) error {
			return node.Realtime(instance)
		}
		testStates[current.COMPLETED] = func(from current.Activity) error { return nil }
		testStates[current.ERROR] = func(from current.Activity) error {
			return node.Error(instance)
		}

		sm := state.NewMachine(testStates)

		instance, _ = internal.CreateServerInstance(defsLst[i], impl, sm, true)
		err := instance.GetConsensus().UpdateNodeConnections()
		if err != nil {
			t.Errorf("Failed to update node connections for node %d: %+v", i, err)
		}

		if errorPhase {
			if i == 0 {
				gc := services.NewGraphGenerator(4, node.GetDefaultPanicHanlder(instance),
					uint8(runtime.NumCPU()), 1, 0)
				g := graphs.InitErrorGraph(gc)
				th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
					return errors.New("Failed intentionally")
				}
				overrides := map[int]phase.Phase{}
				p := phase.New(phase.Definition{
					Graph:               g,
					Type:                phase.PrecompGeneration,
					TransmissionHandler: th,
					Timeout:             500,
					DoVerification:      false,
				})
				overrides[0] = p
				instance.OverridePhases(overrides, t)
			}
			errWg.Add(1)
			f := func(s string) {
				errWg.Done()
			}
			instance.OverridePanicWrapper(f, t)
		}
		instance.RecoveredErrorFilePath = fmt.Sprintf("/tmp/err_%d", i)

		instances = append(instances, instance)
	}

	t.Logf("Initilizing Network for %v nodes", numNodes)
	// initialize the network for every instance
	for _, instance := range instances {
		instance.GetNetwork().DisableAuth()
		instance.Online = true
		_, err := instance.GetNetwork().AddHost(id.PERMISSIONING, testUtil.NDF.Registration.Address,
			[]byte(testUtil.RegCert), false, false)
		if err != nil {
			t.Errorf("Failed to add permissioning host: %v", err)
		}

	}

	t.Logf("Running the Queue for %v nodes", numNodes)
	//begin every instance
	wg := sync.WaitGroup{}
	for _, instance := range instances {
		wg.Add(1)
		localInstance := instance
		go func() {
			time.Sleep(2 * time.Second)
			err := localInstance.Run()
			if err != nil {
				t.Errorf("uh-oh spaghetti-O's: %+v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// Build topology
	ourTopology := make([]string, 0)
	for _, nodeInstance := range instances {
		ourTopology = append(ourTopology, nodeInstance.GetID().String())
	}

	// Construct round info message
	roundInfoMsg := &mixmessages.RoundInfo{
		ID:        0,
		UpdateID:  0,
		State:     uint32(current.PRECOMPUTING),
		BatchSize: uint32(batchsize),
		Topology:  ourTopology,
	}

	expectedbatch, ecrbatch, err := buildMockBatch(batchsize, grp, baseKeys, userID, roundInfoMsg)
	if err != nil {
		t.Errorf("%+v", err)
	}

	done := make(chan struct{})

	go iterate(done, instances, t, ecrbatch, roundInfoMsg, errorPhase)
	<-done
	//wait for last node to be ready to receive the batch
	completedBatch := &mixmessages.Batch{Slots: make([]*mixmessages.Slot, 0)}

	cr, err := instances[numNodes-1].GetCompletedBatchQueue().Receive()
	if err != nil && !strings.Contains(err.Error(), "Did not recieve a completed round") {
		t.Errorf("Unable to receive from CompletedBatchQueue: %+v", err)
	}
	if cr != nil {
		completedBatch.Slots = cr.Round
	}
	//---BUILD PROBING TOOLS----------------------------------------------------

	//get round buffers for probing
	var roundBufs []*round.Buffer
	for _, instance := range instances {
		r, _ := instance.GetRoundManager().GetRound(0)
		roundBufs = append(roundBufs, r.GetBuffer())
	}

	//build i/o map of permutations
	permutationMapping := make([]uint32, batchsize)
	for i := uint32(0); i < uint32(batchsize); i++ {
		slotIndex := i
		for _, buf := range roundBufs {
			slotIndex = buf.Permutations[slotIndex]
		}
		permutationMapping[i] = slotIndex
	}

	//---CHECK OUTPUTS----------------------------------------------------------
	found := 0

	for i := 0; i < batchsize; i++ {

		inputSlot := expectedbatch.Slots[i]
		outputSlot := completedBatch.Slots[permutationMapping[i]]

		success := true

		if grp.NewIntFromBytes(inputSlot.PayloadA).Cmp(grp.NewIntFromBytes(outputSlot.PayloadA)) != 0 {
			t.Errorf("Input slot %v permuted to slot %v payload A did "+
				"not match; \n Expected: %s \n Recieved: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.PayloadA).Text(16),
				grp.NewIntFromBytes(outputSlot.PayloadA).Text(16))
			success = false
		}

		if grp.NewIntFromBytes(inputSlot.PayloadB).Cmp(grp.NewIntFromBytes(outputSlot.PayloadB)) != 0 {
			t.Errorf("Input slot %v permuted to slot %v payload B did "+
				"not match; \n Expected: %s \n Recieved: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.PayloadB).Text(16),
				grp.NewIntFromBytes(outputSlot.PayloadB).Text(16))
			success = false
		}

		if success {
			found++
		}
	}

	if found < batchsize {
		t.Errorf("%v/%v of messages came out incorrect",
			batchsize-found, batchsize)
	} else {
		t.Logf("All messages recieved, passed")
	}

	//---CHECK PRECOMPUTATION---------------------------------------------------

	//SHARE PHASE=
	pk := roundBufs[0].CypherPublicKey.DeepCopy()
	//test that all nodes have the same PK
	for itr, buf := range roundBufs {
		pkNode := buf.CypherPublicKey.DeepCopy()
		if pkNode.Cmp(pk) != 0 {
			t.Errorf("Multinode instance test: node %v does not have "+
				"the same CypherPublicKey as node 1; node 1: %s, node %v: %s",
				itr+1, pk.Text(16), itr+1, pkNode.Text(16))
		}
	}

	//test that the PK is the composition of the Zs
	for _, buf := range roundBufs {
		Z := buf.Z.DeepCopy()
		pkOld := pk.DeepCopy()
		grp.RootCoprime(pkOld, Z, pk)
	}

	if pk.GetLargeInt().Cmp(grp.GetG()) != 0 {
		t.Errorf("Multinode instance test: inverse PK is not equal "+
			"to generator: Expected: %s, Recieved: %s",
			grp.GetG().Text(16), roundBufs[0].CypherPublicKey.Text(16))
	}

	//Final result
	//Traverse the nodes to find the final precomputation for each slot

	//create precomp buffer
	payloadAPrecomps := make([]*cyclic.Int, batchsize)
	payloadBPrecomps := make([]*cyclic.Int, batchsize)

	for i := 0; i < batchsize; i++ {
		payloadAPrecomps[i] = grp.NewInt(1)
		payloadBPrecomps[i] = grp.NewInt(1)
	}

	//precomp Decrypt
	for i := uint32(0); i < uint32(batchsize); i++ {
		for _, buf := range roundBufs {
			grp.Mul(payloadAPrecomps[i], buf.R.Get(i), payloadAPrecomps[i])
			grp.Mul(payloadBPrecomps[i], buf.U.Get(i), payloadBPrecomps[i])
		}
	}

	//precomp permute
	for i := uint32(0); i < uint32(batchsize); i++ {
		slotIndex := i
		for _, buf := range roundBufs {
			grp.Mul(payloadAPrecomps[i], buf.S.Get(slotIndex), payloadAPrecomps[i])
			grp.Mul(payloadBPrecomps[i], buf.V.Get(slotIndex), payloadBPrecomps[i])
			slotIndex = buf.Permutations[slotIndex]
		}
		grp.Inverse(payloadAPrecomps[i], payloadAPrecomps[i])
		grp.Inverse(payloadBPrecomps[i], payloadBPrecomps[i])
	}

	for i := 0; i < batchsize; i++ {
		resultPayloadA := roundBufs[len(roundBufs)-1].PayloadAPrecomputation.Get(permutationMapping[i])
		if payloadAPrecomps[i].Cmp(resultPayloadA) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadA slot %v "+
				"incorrect; Expected: %s, Recieved: %s", i,
				payloadAPrecomps[i].Text(16), resultPayloadA.Text(16))
		}
		resultPayloadB := roundBufs[len(roundBufs)-1].PayloadBPrecomputation.Get(permutationMapping[i])
		if payloadBPrecomps[i].Cmp(resultPayloadB) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadB slot %v "+
				"incorrect; Expected: %s, Recieved: %s", i,
				payloadBPrecomps[i].Text(16), resultPayloadB.Text(16))
		}
	}
}

// buildMockBatch
func buildMockBatch(batchsize int, grp *cyclic.Group, baseKeys []*cyclic.Int,
	userID *id.User, ri *mixmessages.RoundInfo) (*pb.Batch, *pb.Batch, error) {
	//build a batch to send to first node
	expectedbatch := &mixmessages.Batch{}
	ecrbatch := &mixmessages.Batch{}

	kmacHash, err2 := hash.NewCMixHash()
	if err2 != nil {
		return &pb.Batch{}, &pb.Batch{}, errors.Errorf("Could not get KMAC hash: %+v", err2)
	}
	for i := 0; i < batchsize; i++ {
		//make the salt
		salt := make([]byte, 32)
		binary.BigEndian.PutUint64(salt[0:8], uint64(100+6*i))

		//make the payload
		payloadA := grp.NewIntFromUInt(uint64(1 + i)).LeftpadBytes(format.PayloadLen)
		payloadB := grp.NewIntFromUInt(uint64((513 + i) * 256)).LeftpadBytes(format.PayloadLen)

		//make the message
		msg := format.NewMessage()
		msg.SetPayloadA(payloadA)
		msg.SetPayloadB(payloadB)

		//encrypt the message
		ecrMsg := cmix.ClientEncrypt(grp, msg, salt, baseKeys)
		kmacs := cmix.GenerateKMACs(salt, baseKeys, kmacHash)

		//make the slot
		ecrslot := &mixmessages.Slot{}
		ecrslot.PayloadA = ecrMsg.GetPayloadA()
		ecrslot.PayloadB = ecrMsg.GetPayloadB()
		ecrslot.SenderID = userID.Bytes()
		ecrslot.Salt = salt
		ecrslot.KMACs = kmacs

		ecrbatch.Slots = append(ecrbatch.Slots, ecrslot)
		ecrbatch.Round = ri

		slot := &mixmessages.Slot{}
		slot.PayloadA = msg.GetPayloadA()
		slot.PayloadB = msg.GetPayloadB()
		slot.SenderID = userID.Bytes()
		slot.Salt = salt
		expectedbatch.Slots = append(expectedbatch.Slots, slot)
	}

	return expectedbatch, ecrbatch, nil
}

//
func iterate(done chan struct{}, nodes []*internal.Instance, t *testing.T,
	ecrBatch *pb.Batch, roundInfoMsg *mixmessages.RoundInfo, errorPhase bool) {
	// Define a mechanism to wait until the next state
	asyncWaitUntil := func(wg *sync.WaitGroup, until current.Activity, node *internal.Instance) {
		wg.Add(1)
		go func() {
			success, err := node.GetStateMachine().WaitForUnsafe(until, 5*time.Second, t)
			//			t.Logf("success: %+v\nerr: %+v\n stateMachine: %+v", success, err, node.GetStateMachine())
			if !success {
				jww.FATAL.Printf("Wait for node %s to enter state %s failed: %s", node.GetID(), until.String(), err)
			} else {
				wg.Done()
			}
		}()

	}

	//wait until all nodes are started
	wg := sync.WaitGroup{}

	// Parse through the nodes prepping them for rounds
	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.WAITING, nodeInstance)
	}

	wg.Wait()
	// Mocking permissioning server signing message
	signRoundInfo(roundInfoMsg)

	for index, nodeInstance := range nodes {
		err := nodeInstance.GetConsensus().RoundUpdate(roundInfoMsg)
		if err != nil {
			t.Errorf("Failed to updated network instance for new round info: %v", err)
		}

		// Send the round info to the instance (to be handled internally later)
		err = nodeInstance.GetCreateRoundQueue().Send(roundInfoMsg)
		if err != nil {
			t.Errorf("Unable to send to RealtimeRoundQueue for node %d: %+v", index, err)
		}

		// Begin the PRECOMPUTING state
		ok, err := nodeInstance.GetStateMachine().Update(current.PRECOMPUTING)
		if !ok || err != nil {
			t.Errorf("Cannot move to precomputing state: %+v", err)
		}

	}

	if errorPhase {
		errWg.Wait()
		done <- struct{}{}
		return
	}

	// need to look in permissioning, manually do steps
	// Parse through the nodes prepping them for rounds
	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.STANDBY, nodeInstance)
	}

	wg.Wait()
	for _, nodeInstance := range nodes {
		// Send info to the realtime round queue
		err := nodeInstance.GetRealtimeRoundQueue().Send(roundInfoMsg)
		if err != nil {
			jww.FATAL.Printf("Unable to send to RealtimeRoundQueue: %+v", err)
		}

		ok, err := nodeInstance.GetStateMachine().Update(current.REALTIME)
		if !ok || err != nil {
			jww.FATAL.Printf("Failed to update to realtime: %+v", err)
		}
	}

	err := io.HandleRealtimeBatch(nodes[0], ecrBatch, io.PostPhase)
	if err != nil {
		t.Errorf("Unable to handle realtime batch: %+v", err)
	}

	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.COMPLETED, nodeInstance)
	}

	wg.Wait()
	done <- struct{}{}
}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)
	return nil
}

func makeMultiInstanceParams(numNodes, portstart int, useGPU bool) []*internal.Definition {

	//generate IDs and addresses
	var nidLst []*id.Node
	var nodeLst []internal.Node
	addrFmt := "localhost:%03d"
	for i := 0; i < numNodes; i++ {
		//generate id
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nidLst = append(nidLst, nodeID)
		//generate address
		addr := fmt.Sprintf(addrFmt, i+portstart)

		n := internal.Node{
			ID:      nodeID,
			Address: addr,
		}
		nodeLst = append(nodeLst, n)

	}

	networkDef := buildNdf(nodeLst)

	//generate parameters list
	var defLst []*internal.Definition

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	for i := 0; i < numNodes; i++ {

		def := internal.Definition{
			ID: nidLst[i],
			Flags: internal.Flags{
				KeepBuffers: true,
				UseGPU:      useGPU,
			},
			TlsCert: []byte(testUtil.RegCert),
			Gateway: internal.GW{
				ID:      nidLst[i].NewGateway(),
				TlsCert: nil,
				Address: "",
			},
			UserRegistry:    &globals.UserMap{},
			ResourceMonitor: &measure.ResourceMonitor{},
			FullNDF:         networkDef,
			PartialNDF:      networkDef,
			Address:         nodeLst[i].Address,
			MetricsHandler:  func(i *internal.Instance, roundID id.Round) error { return nil },
			GraphGenerator:  services.NewGraphGenerator(4, PanicHandler, 1, 4, 1.0),
			RngStreamGen: fastRNG.NewStreamGenerator(10000,
				uint(runtime.NumCPU()), csprng.NewSystemRNG),
			RoundCreationTimeout: 2,
		}

		defLst = append(defLst, &def)
	}

	return defLst
}

func makeMultiInstanceGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	return cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))
}

// buildNdf builds the ndf used for definitions
func buildNdf(nodeLst []internal.Node) *ndf.NetworkDefinition {
	// Pull the node id's out of nodeList
	ndfNodes := make([]ndf.Node, 0)
	for _, ourNode := range nodeLst {
		tmpNode := ndf.Node{
			ID:             ourNode.ID.Bytes(),
			Address:        ourNode.Address,
			TlsCertificate: string(ourNode.TlsCert),
		}
		ndfNodes = append(ndfNodes, tmpNode)

	}

	// Build a group
	group := ndf.Group{
		Prime: "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF",
		SmallPrime: "2",
		Generator:  "2",
	}

	// Construct an ndf
	return &ndf.NetworkDefinition{
		Timestamp: time.Time{},
		Nodes:     ndfNodes,
		E2E:       group,
		CMIX:      group,
	}

}
