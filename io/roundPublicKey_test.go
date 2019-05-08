////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// FIXME: this import list makes it feel like the api is spaghetti
import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"testing"
	"time"
)

var nodeIDsPk *services.NodeIDList
var instancesPk []*server.Instance

func TestPostRoundPublicKey_Transmit(t *testing.T) {

	testPhase := phase.New(InitMockGraph(services.
		NewGraphGenerator(1, nil, 1, 1, 1)),
		phase.RealPermute, TransmitPhase, time.Second)

	// Now fix the round manager
	rm := instancesPk[2].GetRoundManager()
	roundID := id.Round(42)

	phases := make([]*phase.Phase, 1)
	phases[0] = testPhase

	thisRound := round.New(grp, roundID, phases, nil, 0, 1)
	rm.AddRound(thisRound)

	roundPubKey := grp.NewIntFromUInt(42)

	err := TransmitRoundPublicKey(instancesPk[0].GetNetwork(),roundPubKey, 42,
		nodeIDsPk)

	// TODO: Cycle through all the servers and ensure the
	// roundPublicKey is set to the same value.
	if err != nil {
		t.Errorf("%v", err)
	}
}

func TestPostRoundPublicKey_SetsRoundBuff(t *testing.T) {
	grp = cyclic.NewGroup(large.NewIntFromString(primeString, 16),
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
	grp = cyclic.NewGroup(large.NewIntFromString(primeString, 16),
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
	if err != node.ErrOutsideOfGroup {
		t.Errorf("PostRoundPublic key did not return an outside of group error")
	}

}
