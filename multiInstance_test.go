package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
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
			Slots:    make([]*mixmessages.Slot, localBatchSize),
			ForPhase: int32(phase.PrecompGeneration),
			Round: &mixmessages.RoundInfo{
				ID: uint64(rid),
			},
		}
		for i := 0; i < int(localBatchSize); i++ {
			newBatch.Slots[i] = &mixmessages.Slot{}
		}

		node.ReceivePostPhase(newBatch, instance)
		return nil
	}

	firstNode.InitFirstNode()
	firstNode.RunFirstNode(firstNode, 10*time.Second,
		io.TransmitCreateNewRound, starter)

	//build a batch to send to first node
	newbatch := mixmessages.Batch{}
	for i := 0; i < batchsize; i++ {
		//make the salt
		salt := make([]byte, 64)
		binary.BigEndian.PutUint64(salt[0:8], uint64(100+6*i))

		//make the payload
		payload := make([]byte, format.TOTAL_LEN)
		binary.BigEndian.PutUint64(payload[0:8], uint64(500+12*i))

		//make the message
		msg := format.NewMessage()
		msg.Payload.SetPayload(payload)

		//encrypt the message
		ecrMsg := cmix.ClientEncryptDecrypt(true, grp, msg, salt, baseKeys)

		//make the slot
		slot := &mixmessages.Slot{}
		slot.MessagePayload = ecrMsg.SerializePayload()
		slot.AssociatedData = ecrMsg.SerializeAssociatedData()
		slot.SenderID = userID.Bytes()
		slot.Salt = salt
		newbatch.Slots = append(newbatch.Slots, slot)
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

	found := 0

	for i := 0; i < batchsize; i++ {
		completedSlot := completedBatch.Slots[i]
		for j := 0; j < batchsize; j++ {
			if reflect.DeepEqual(completedSlot.MessagePayload, newbatch.Slots[i].MessagePayload) {
				found++
				break
			}
		}
	}

	if found < batchsize {
		var expectedPayloads string
		for _, slot := range newbatch.Slots {
			expectedPayloads += "   " +
				base64.StdEncoding.EncodeToString(slot.MessagePayload) + "\n"
		}

		var recievedPayloads string
		for _, slot := range completedBatch.Slots {
			recievedPayloads += "   " +
				base64.StdEncoding.EncodeToString(slot.MessagePayload) + "\n"
		}

		t.Errorf("Multinode instance test: not all messages "+
			"came out correctly\n Expected: \n %s Recieved: \n %s",
			expectedPayloads, recievedPayloads)
	}
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
