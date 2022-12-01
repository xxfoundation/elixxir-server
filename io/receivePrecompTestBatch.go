////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"strings"
	"time"
)

// ReceivePrecompTestBatch is a streaming reception handler which receives a
// test batch of random data from the last node in order to verify the data
// can be sent over the connection because a similar set fo data will be
// sent on the last leg of realtime. It will denote in the round object
// if the transmission was successful. Is called by TransmitPrecompTestBatch.
func ReceivePrecompTestBatch(instance *internal.Instance, stream pb.Node_PrecompTestBatchServer, message *pb.RoundInfo, auth *connect.Auth) error {
	//check that the round is in the correct state to receive this transmission
	curActivity, err := instance.GetStateMachine().WaitFor(5*time.Second, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	//Get the correct round
	roundID := id.Round(message.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	// Check for proper authentication
	// and if the sender is the last member of the round
	topology := r.GetTopology()
	senderId := auth.Sender.GetId()
	if !auth.IsAuthenticated || !topology.GetLastNode().Cmp(senderId) {
		jww.WARN.Printf("ReceivePrecompTestBatch Error: "+
			"Attempted communication by %+v has not been authenticated or "+
			"is not last node: %s", auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	nodeID := instance.GetID()
	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeID, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeID),
		topology.Len())

	jww.INFO.Printf("[%s] RID %d ReceivePrecompTestBatch "+
		"START", name, roundID)

	grp := instance.GetNetworkStatus().GetCmixGroup()

	expectedDataSize := 2 * grp.GetP().ByteLen() * int(r.GetBatchSize())

	// Receive the slots
	slot, err := stream.Recv()
	size := 0
	slotsReceived := uint32(0)
	for ; err == nil; slot, err = stream.Recv() {
		slotsReceived++
		size += len(slot.PayloadA) + len(slot.PayloadB)
	}
	errClose := stream.SendAndClose(&messages.Ack{})
	if errClose != nil {
		return errors.Errorf("Failed to close stream for round %d: %v", roundID, err)
	}

	//check the reception came through correctly
	if err != io.EOF {
		err = errors.Errorf("error occurred on PrecompTestBatch round %v, %v/%v slots "+
			"recived: %s", roundID, slotsReceived, r.GetBatchSize(), err.Error())
		jww.ERROR.Printf("%s", err.Error())
		return err
	} else if slotsReceived != r.GetBatchSize() {
		err = errors.Errorf("error occurred on PrecompTestBatch round %v, incorrect "+
			"number of slots received, %v slots received vs %v slots expected", roundID, slotsReceived,
			r.GetBatchSize())
		jww.ERROR.Printf("%s", err.Error())
		return err
	} else if size != expectedDataSize {
		err = errors.Errorf("error occurred on PrecompTestBatch round %v, incorrect "+
			"data size received, %v/%v slots, %v bytes vs %v bytes", roundID, slotsReceived,
			r.GetBatchSize(), size, expectedDataSize)
		jww.ERROR.Printf("%s", err.Error())
		return err
	}

	jww.INFO.Printf("[%s] RID %d ReceivePrecompTestBatch "+
		"COMPLETE", name, roundID)

	//denote that the reception was successful so that precomp can complete
	r.DenotePrecompBroadcastSuccess()

	return nil
}
