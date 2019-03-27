////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/globals"
	"time"
)

var GetRoundBufferInfoTimeout = "1s"

// Start round receives a list of CmixMessages and sends them to the
// ReceiveMessageFromClient handler.
func StartRound(messages *pb.InputMessages) {
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
		ReceiveMessageFromClient(cMixMsgs[i])
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished StartRound(...) in %d ms",
		(endTime.Sub(startTime))/time.Millisecond)
}

// GetRoundBufferInfo returns # of completed precomputations
func GetRoundBufferInfo() (int, error) {
	tout, _ := time.ParseDuration(GetRoundBufferInfoTimeout)
	c := make(chan error, 1)
	go func() {
		for len(RoundCh) == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		c <- nil
	}()
	select {
	case _ = <-c:
		return len(RoundCh), nil
	case <-time.After(tout):
		return len(RoundCh), errors.New("round buffer is empty")
	}
}
