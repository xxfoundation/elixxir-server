package main

import (
	"encoding/binary"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/conf"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"reflect"
	"sync"
	"testing"
	"time"
)

func Test_MultiInstance_N3_B8(t *testing.T) {
	MultiInstanceTest(3, 8, t)
}

func MultiInstanceTest(numNodes, batchsize int, t *testing.T) {

	jww.SetStdoutThreshold(jww.LevelInfo)

	if numNodes < 3 {
		t.Errorf("Multi Instance Test must have a minnimum of 3 nodes,"+
			" Recieved %v", numNodes)
	}

	grp := makeMultiInstanceGroup()

	//get parameters
	paramLst := makeMultiInstanceParams(numNodes, batchsize, 1000, grp)

	//make user for sending messages
	userID := id.NewUserFromUint(42, t)
	var baseKeys []*cyclic.Int
	for i := 0; i < numNodes; i++ {
		baseKey := grp.NewIntFromUInt(uint64(1000 + 5*i))
		baseKeys = append(baseKeys, baseKey)
	}

	//build the registries for every node
	var registries []globals.UserRegistry

	for i := 0; i < numNodes; i++ {
		var registry globals.UserRegistry
		registry = &globals.UserMap{}
		user := globals.User{
			ID:      userID,
			BaseKey: baseKeys[i],
		}
		registry.UpsertUser(&user)
		registries = append(registries, registry)
	}

	//build the instances
	var instances []*server.Instance

	t.Logf("Building instances for %v nodes", numNodes)

	for i := 0; i < numNodes; i++ {
		instance := server.CreateServerInstance(paramLst[i], registries[i],
			nil, nil)
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
		go func() {
			localInstance.InitNetwork(node.NewImplementation)
			wg.Done()
		}()
	}

	wg.Wait()

	t.Logf("Running the Queue for %v nodes", numNodes)
	//begin every instance
	for _, instance := range instances {
		instance.Run()
	}

	t.Logf("Initalizing the first node, begining operations")
	//Initialize the first node
	localBatchSize := batchsize
	starter := func(instance *server.Instance, rid id.Round) error {
		newBatch := &mixmessages.Batch{
			Slots:     make([]*mixmessages.Slot, localBatchSize),
			FromPhase: int32(phase.PrecompGeneration),
			Round: &mixmessages.RoundInfo{
				ID: uint64(rid),
			},
		}
		for i := 0; i < int(localBatchSize); i++ {
			newBatch.Slots[i] = &mixmessages.Slot{}
		}

		//get the round from the instance
		rm := instance.GetRoundManager()
		r, err := rm.GetRound(rid)

		if err != nil {
			jww.CRITICAL.Panicf("First Node Round Init: Could not get "+
				"round (%v) right after round init", rid)
		}

		//get the phase
		p := r.GetCurrentPhase()

		//queue the phase to be operated on if it is not queued yet
		p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		//send the data to the phase
		err = io.PostPhase(p, newBatch)

		if err != nil {
			jww.ERROR.Panicf("Error first node generation init: "+
				"should be able to return: %+v", err)
		}
		return nil
	}

	firstNode.InitFirstNode()
	firstNode.RunFirstNode(firstNode, 10*time.Second,
		io.TransmitCreateNewRound, starter)

	lastNode.InitLastNode()

	//build a batch to send to first node
	newbatch := mixmessages.Batch{}
	for i := 0; i < batchsize; i++ {
		//make the salt
		salt := make([]byte, 32)
		binary.BigEndian.PutUint64(salt[0:8], uint64(100+6*i))

		//make the payload
		payload := grp.NewIntFromUInt(uint64(1 + i)).LeftpadBytes(32)
		fmt.Println(payload)

		//make the message
		msg := format.NewMessage()
		msg.Payload.SetPayload(payload)
		msg.AssociatedData.SetRecipientID(salt)

		//encrypt the message
		//		ecrMsg := cmix.ClientEncryptDecrypt(true, grp, msg, salt, baseKeys)

		//make the slot
		slot := &mixmessages.Slot{}
		slot.MessagePayload = msg.SerializePayload()
		slot.AssociatedData = msg.SerializeAssociatedData()
		slot.SenderID = userID.Bytes()
		slot.Salt = salt
		newbatch.Slots = append(newbatch.Slots, slot)
	}

	//wait until the first node is ready for a batch
	numPrecompsAvalible := 0
	var err error

	for numPrecompsAvalible == 0 {
		numPrecompsAvalible, err = io.GetRoundBufferInfo(firstNode.GetCompletedPrecomps(), 100*time.Millisecond)
		if err != nil && err != io.Err_EmptyRoundBuff {
			t.Errorf("MultiNode Test: Error returned from first node "+
				"`GetRoundBufferInfo`: %v", err)
		}
	}

	//send the batch to the node
	err = node.ReceivePostNewBatch(firstNode, &newbatch)

	if err != nil {
		t.Errorf("MultiNode Test: Error returned from first node "+
			"`ReceivePostNewBatch`: %v", err)
	}

	//wait for last node to be ready to receive the batch
	var completedBatch *mixmessages.Batch
	err = io.Err_NoCompletedBatch
	for err == io.Err_NoCompletedBatch {
		completedBatch, err = io.GetCompletedBatch(lastNode.GetCompletedBatchQueue(), 100*time.Millisecond)
	}

	//---BUILD PROBING TOOLS----------------------------------------------------

	//get round buffers for probing
	var roundBufs []*round.Buffer
	for _, instance := range instances {
		roundBufs = append(roundBufs, getRoundBuf(instance, 0))
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

		inputSlot := newbatch.Slots[i]
		outputSlot := completedBatch.Slots[permutationMapping[i]]

		success := true

		if !reflect.DeepEqual(inputSlot.MessagePayload, outputSlot.MessagePayload) {
			t.Errorf("Input slot %v permuted to slot %v payload A did "+
				"not match; \n Expected: %s \n Recieved: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.MessagePayload).Text(16),
				grp.NewIntFromBytes(outputSlot.MessagePayload).Text(16))
			success = false
		}

		if !reflect.DeepEqual(inputSlot.AssociatedData, outputSlot.AssociatedData) {
			t.Errorf("Input slot %v permuted to slot %v payload B did "+
				"not match; \n Expected: %s \n Recieved: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.AssociatedData).Text(16),
				grp.NewIntFromBytes(outputSlot.AssociatedData).Text(16))
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

	//SHARE PHASE

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
			grp.Mul(payloadAPrecomps[i], buf.R.Get(slotIndex), payloadAPrecomps[i])
			grp.Mul(payloadBPrecomps[i], buf.U.Get(slotIndex), payloadBPrecomps[i])
			slotIndex = buf.Permutations[slotIndex]
		}
		grp.Inverse(payloadAPrecomps[i], payloadAPrecomps[i])
		grp.Inverse(payloadBPrecomps[i], payloadBPrecomps[i])
	}

	for i := 0; i < batchsize; i++ {
		resultPayloadA := roundBufs[len(roundBufs)-1].MessagePrecomputation.Get(permutationMapping[i])
		if payloadAPrecomps[i].Cmp(resultPayloadA) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadA slot %v "+
				"incorrect; Expected: %s, Recieved: %s", i,
				payloadAPrecomps[i].Text(16), resultPayloadA.Text(16))
		}
		resultPayloadB := roundBufs[len(roundBufs)-1].ADPrecomputation.Get(permutationMapping[i])
		if payloadBPrecomps[i].Cmp(resultPayloadB) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadB slot %v "+
				"incorrect; Expected: %s, Recieved: %s", i,
				payloadBPrecomps[i].Text(16), resultPayloadB.Text(16))
		}
	}
}

func getRoundBuf(instance *server.Instance, id id.Round) *round.Buffer {
	r, _ := instance.GetRoundManager().GetRound(0)
	return r.GetBuffer()
}

func makeMultiInstanceParams(numNodes, batchsize, portstart int, grp *cyclic.Group) []*conf.Params {

	//generate IDs and addresses
	var nidLst []string
	var addrLst []string
	addrFmt := "localhost:5%03d"
	for i := 0; i < numNodes; i++ {
		//generate id
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nidLst = append(nidLst, nodeID.String())
		//generate address
		addr := fmt.Sprintf(addrFmt, i+portstart)
		addrLst = append(addrLst, addr)
	}

	//generate parameters list
	var paramsLst []*conf.Params

	for i := 0; i < numNodes; i++ {

		param := conf.Params{
			Groups: conf.Groups{
				CMix: grp,
			},
			NodeAddresses: addrLst,
			NodeIDs:       nidLst,
			Batch:         uint32(batchsize),
			Index:         i,
		}

		paramsLst = append(paramsLst, &param)
	}

	return paramsLst
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
		large.NewInt(2), large.NewInt(1283))
}
