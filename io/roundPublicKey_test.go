////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"sync"
	"testing"
)

const primeString =
	"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var receivedPk *mixmessages.RoundPublicKey
var donePostRoundPk sync.Mutex

func TestPostRoundPublicKey_Transmit(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]func() *node.Implementation{nil, mockPostRoundPKImplementation, mockPostRoundPKImplementation},
		)
	defer Shutdown(comms)

	// Build the mock functions called by the transmitter
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(1283))
	roundPubKey := grp.NewIntFromUInt(42)
	roundID := id.Round(5)

	ids := []*id.Node{
		topology.GetNodeAtIndex(1),
		topology.GetNodeAtIndex(2),
	}

	donePostRoundPk.Lock()

	//call the transmitter
	err := TransmitRoundPublicKey(comms[0], roundPubKey, roundID,
		topology, ids)

	if err != nil {
		t.Errorf("TransmitPhase: Unexpected error: %+v", err)
	}

	//Use lock to wait until handler receives results
	donePostRoundPk.Lock()
	defer donePostRoundPk.Unlock()

	//testPhase := phase.New(InitMockGraph(services.
	//	NewGraphGenerator(1, nil, 1, 1, 1)),
	//	phase.RealPermute, TransmitPhase, time.Second)
	//
	//// Now fix the round manager
	//rm := instancesPk[2].GetRoundManager()
	//roundID := id.Round(42)
	//
	//phases := make([]*phase.Phase, 1)
	//phases[0] = testPhase
	//
	//thisRound := round.New(grp, roundID, phases, nil, 0, 1)
	//rm.AddRound(thisRound)
	//

	// TODO: Cycle through all the servers and ensure the
	// roundPublicKey is set to the same value.
	if err != nil {
		t.Errorf("%v", err)
	}
}

func TestPostRoundPublicKey_SetsRoundBuff(t *testing.T) {

	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(1283))

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
		large.NewInt(2), large.NewInt(97))

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
		large.NewInt(3), large.NewInt(43))
	key = grp.NewMaxInt()
	pk = mixmessages.RoundPublicKey{Key: key.Bytes()}

	err = PostRoundPublicKey(grp2, roundBuff, &pk)

	// Ensure it does not return an error
	if err != services.ErrOutsideOfGroup {
		t.Errorf("PostRoundPublic key did not return an outside of group error")
	}

}

func mockPostRoundPKImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {
		receivedPk = pk
		done.Unlock()
	}
	return impl
}
