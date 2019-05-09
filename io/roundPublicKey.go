////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// TransmitRoundPublicKey sends the public key to every node
// in the round
func TransmitRoundPublicKey(network *node.NodeComms, pubKey *cyclic.Int, roundID id.Round,
	topology *circuit.Circuit, ids []*id.Node ) error {

	// Create the message structure to send the messages
	roundPubKeyMsg := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		Key: pubKey.Bytes(),
	}


	// Send public key to all nodes
	for index :=0; index < topology.Len(); index++ {

		receipient := topology.GetNodeAtIndex(index)
		if topology.IsFirstNode(receipient) {
			continue
		}

		ack, err := network.SendPostRoundPublicKey(receipient, roundPubKeyMsg)

		// Make sure the comm doesn't return an Ack with an
		// error message
		if ack != nil && ack.Error != "" {
			err = errors.Errorf("Remote Server Error: %s, %s",
				receipient, ack.Error)
			return err
		}
	}

	return nil
}

// PostRoundPublicKey implements the server gRPC reception handler
// for posting a public key to the round Transmission handler
func PostRoundPublicKey(grp *cyclic.Group, roundBuff *round.Buffer, pk *mixmessages.RoundPublicKey) error {

	inside := grp.BytesInside(pk.GetKey())

	if !inside {
		return services.ErrOutsideOfGroup
	}

	grp.SetBytes(roundBuff.CypherPublicKey, pk.GetKey())

	return nil
}
