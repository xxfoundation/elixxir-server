////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

const primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var receivedPks [1]*mixmessages.RoundPublicKey

func TestPostRoundPublicKey_Transmit(t *testing.T) {
	//fixme: modify test to check multiple nodes
	// need to set up multiple instances, set topology, and add hosts to topology and to instances

	instance, nodeAddr := mockInstance(t, mockPostRoundPKImplementation0)

	// Build the mock functions called by the transmitter
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))
	roundPubKey := grp.NewIntFromUInt(42)

	roundID := id.Round(5)
	chunkCnt := uint32(0)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompDecrypt,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompDecrypt})

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompDecrypt
	responseMap := make(phase.ResponseMap)
	responseMap["PrecompDecrypt"] = response

	topology := connect.NewCircuit([]*id.Node{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID().String(), nodeAddr, cert, false, true)
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID().String(), nodeAddr, cert, false, true)
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")
	if err != nil {
		t.Error()
	}

	instance.GetRoundManager().AddRound(rnd)

	getChunk := func() (services.Chunk, bool) {
		if chunkCnt == 0 {
			chunk, ok := services.NewChunk(chunkCnt, chunkCnt+1), true
			chunkCnt++
			return chunk, ok
		}
		return services.NewChunk(0, 0), false
	}

	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{PartialRoundPublicCypherKey: roundPubKey.Bytes()}
	}

	//call the transmitter
	err = TransmitRoundPublicKey(roundID, instance, getChunk, getMsg)

	if err != nil {
		t.Errorf("TransmitRoundPublicKey: Unexpected error: %+v", err)
	}

	expected := roundPubKey

	// Ensure the roundPublicKey is set to the correct value for every
	// recipient
	for index, pk := range receivedPks {
		actual := grp.NewIntFromBytes(pk.Key)

		if expected.Cmp(actual) != 0 {
			t.Errorf("TransmitRoundPublicKey: Incorrect public key from node %v"+
				"Expected: %v, Recieved: %v", index, expected, actual)
		}
	}
}

func TestPostRoundPublicKey_SetsRoundBuff(t *testing.T) {

	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	// Initialize round buffer
	batchSize := uint32(100)
	expandedBatchSize := uint32(100)
	roundBuff := round.NewBuffer(grp, batchSize, expandedBatchSize)

	// Initialize public key message
	key := grp.NewInt(123)
	pk := mixmessages.RoundPublicKey{Key: key.Bytes()}

	// Call PostRoundPublic Key
	err := PostRoundPublicKey(grp, roundBuff, &pk)

	// Ensure it does not return an error
	if err != nil {
		t.Errorf("PostRoundPublic key returned an error")
	}

	// Verify public key was set in the round buffer
	if roundBuff.CypherPublicKey.Cmp(key) != 0 {
		t.Errorf("Public key was not set to the correct value")
	}
}

func TestPostRoundPublicKey_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	// Initialize round buffer
	batchSize := uint32(100)
	expandedBatchSize := uint32(100)
	roundBuff := round.NewBuffer(grp, batchSize, expandedBatchSize)

	// Initialize public key message
	key := grp.NewInt(123)
	pk := mixmessages.RoundPublicKey{Key: key.Bytes()}

	// Call PostRoundPublic Key
	err := PostRoundPublicKey(grp, roundBuff, &pk)

	// Ensure it does not return an error
	if err != nil {
		t.Errorf("PostRoundPublic key returned an error")
	}

	// Call PostRoundPublic Key with public key value outside of group
	grp2 := cyclic.NewGroup(large.NewInt(97),
		large.NewInt(3))
	key = grp.NewMaxInt()
	pk = mixmessages.RoundPublicKey{Key: key.Bytes()}

	err = PostRoundPublicKey(grp2, roundBuff, &pk)

	// Ensure it does not return an error
	if err != services.ErrOutsideOfGroup {
		t.Errorf("PostRoundPublic key did not return an outside of group error")
	}

}

func mockPostRoundPKImplementation0(instance *server.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
		receivedPks[0] = pk
		return nil
	}
	return impl
}

//func mockPostRoundPKImplementation1() *node.Implementation {
//	impl := node.NewImplementation()
//	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
//		receivedPks[1] = pk
//		return nil
//	}
//	return impl
//}
//
//func mockPostRoundPKImplementation2() *node.Implementation {
//	impl := node.NewImplementation()
//	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
//		receivedPks[2] = pk
//		return nil
//	}
//	return impl
//}
