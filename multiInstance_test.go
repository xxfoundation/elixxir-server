package main

import (
	"encoding/binary"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"runtime"
	"sync"
	"testing"
	"time"
)

func Test_MultiInstance_N3_B8(t *testing.T) {
	MultiInstanceTest(3, 32, t)
}

func MultiInstanceTest(numNodes, batchsize int, t *testing.T) {

	jww.SetStdoutThreshold(jww.LevelDebug)

	if numNodes < 3 {
		t.Errorf("Multi Instance Test must have a minnimum of 3 nodes,"+
			" Recieved %v", numNodes)
	}

	grp := makeMultiInstanceGroup()

	//get parameters
	defsLst := makeMultiInstanceParams(numNodes, batchsize, 1000, grp)

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

	//build the instances
	var instances []*server.Instance

	t.Logf("Building instances for %v nodes", numNodes)

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	for i := 0; i < numNodes; i++ {
		instance, _ := server.CreateServerInstance(defsLst[i], node.NewImplementation, true)
		instances = append(instances, instance)
	}

	firstNode := instances[0]
	lastNode := instances[len(instances)-1]

	t.Logf("Initilizing Network for %v nodes", numNodes)
	//initialize the network for every instance
	wg := sync.WaitGroup{}
	for _, instance := range instances {
		wg.Add(1)
		localInstance := instance
		jww.INFO.Println("Waiting...")
		go func() {
			localInstance.Online = true
			wg.Done()
		}()
	}

	wg.Wait()

	t.Logf("Running the Queue for %v nodes", numNodes)
	//begin every instance
	for _, instance := range instances {
		io.VerifyServersOnline(instance.GetNetwork(), instance.GetTopology())
		instance.Run()
	}

	t.Logf("Initalizing the first node, begining operations")
	//Initialize the first node

	firstNode.InitFirstNode()
	firstNode.RunFirstNode(firstNode, 10*time.Second,
		io.TransmitCreateNewRound, node.MakeStarter(uint32(batchsize)))

	lastNode.InitLastNode()

	//build a batch to send to first node
	expectedbatch := mixmessages.Batch{}
	ecrbatch := mixmessages.Batch{}

	kmacHash, err2 := hash.NewCMixHash()
	if err2 != nil {
		t.Errorf("Could not get KMAC hash: %+v", err2)
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

		slot := &mixmessages.Slot{}
		slot.PayloadA = msg.GetPayloadA()
		slot.PayloadB = msg.GetPayloadB()
		slot.SenderID = userID.Bytes()
		slot.Salt = salt
		expectedbatch.Slots = append(expectedbatch.Slots, slot)
	}

	//wait until the first node is ready for a batch
	numPrecompsAvalible := 0
	var err error

	for numPrecompsAvalible == 0 {
		numPrecompsAvalible, err = io.GetRoundBufferInfo(firstNode.GetCompletedPrecomps(), 100*time.Millisecond)
		if err != nil {
			t.Errorf("MultiNode Test: Error returned from first node "+
				"`GetRoundBufferInfo`: %v", err)
		}
	}

	h, _ := connect.NewHost(firstNode.GetID().NewGateway().String(), "test", nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	//send the batch to the node
	err = node.ReceivePostNewBatch(firstNode, &ecrbatch, auth)

	if err != nil {
		t.Errorf("MultiNode Test: Error returned from first node "+
			"`ReceivePostNewBatch`: %v", err)
	}

	//wait for last node to be ready to receive the batch
	completedBatch := &mixmessages.Batch{Slots: make([]*mixmessages.Slot, 0)}
	h, _ = connect.NewHost(lastNode.GetID().NewGateway().String(), "test", nil, false, false)
	for len(completedBatch.Slots) == 0 {
		completedBatch, _ = io.GetCompletedBatch(lastNode, 100*time.Millisecond, &connect.Auth{
			IsAuthenticated: true,
			Sender:          h,
		})
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

func makeMultiInstanceParams(numNodes, batchsize, portstart int, grp *cyclic.Group) []*server.Definition {

	//generate IDs and addresses
	var nidLst []*id.Node
	var nodeLst []server.Node
	addrFmt := "localhost:5%03d"
	for i := 0; i < numNodes; i++ {
		//generate id
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nidLst = append(nidLst, nodeID)
		//generate address
		addr := fmt.Sprintf(addrFmt, i+portstart)

		n := server.Node{
			ID:      nodeID,
			Address: addr,
		}
		nodeLst = append(nodeLst, n)

	}

	//generate parameters list
	var defLst []*server.Definition

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	for i := 0; i < numNodes; i++ {

		def := server.Definition{
			CmixGroup: grp,
			Topology:  connect.NewCircuit(nidLst),
			ID:        nidLst[i],
			BatchSize: uint32(batchsize),
			Nodes:     nodeLst,
			Flags: server.Flags{
				KeepBuffers: true,
			},
			Gateway: server.GW{
				ID:      nidLst[i].NewGateway(),
				TlsCert: nil,
				Address: "",
			},
			Address:        nodeLst[i].Address,
			MetricsHandler: func(i *server.Instance, roundID id.Round) error { return nil },
			GraphGenerator: services.NewGraphGenerator(4, PanicHandler, 1, 4, 0.0),
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
