////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
	"time"
)

// Start round receives a list of CmixMessages and sends them to the
// ReceiveMessageFromClient handler.
func (m ServerImpl) StartRound(messages *pb.InputMessages) {
	startTime := time.Now()
	jww.INFO.Printf("Starting StartRound(...) at %s",
		startTime.Format(time.RFC3339))

	cMixMsgs := messages.Messages

	// FIXME: There's no easy way to guarantee the batch sent here will run until
	// we remove direct client comms to the server.
	if uint64(len(cMixMsgs)) != globals.BatchSize {
		// TODO: We should return a failure ack, but that's not in comms yet
		jww.ERROR.Printf("StartRound(...) batch size is %d, but have batch of %d",
			len(cMixMsgs), globals.BatchSize)
	}

	// Keep going even if we have an error.
	for i := range cMixMsgs {
		m.ReceiveMessageFromClient(cMixMsgs[i])
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished StartRound(...) in %d ms",
		(endTime.Sub(startTime))/time.Millisecond)
}
