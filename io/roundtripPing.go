////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"strconv"
	"time"
)

// Records current time and sends all recorded times to next node
func (s ServerImpl) RoundtripPing(msg *pb.TimePing) {
	// record current time
	times := append(msg.Times, time.Now().UnixNano())
	// if not last node, send to next node. otherwise log results
	if !id.IsLastNode {
		node.SendRoundtripPing(Servers[len(times)-1],
			&pb.TimePing{
				Times: times,
			})
	} else {
		LogPingTime(times)
	}
}

// Initiates a roundtrip ping starting at last node
func GetRoundtripPing(servers []string) {
	// if only one node then just print an empty log statement
	if len(servers) > 1 {
		times := make([]int64, 0)
		times = append(times, time.Now().UnixNano())
		node.SendRoundtripPing(servers[0], &pb.TimePing{
			Times: times,
		})
	}
}

// Logs the results of the roundtrip ping in milliseconds between nodes
func LogPingTime(times []int64) {
	stringTimes := strconv.FormatInt((times[1]-times[0])/1000000, 10)
	for i := 2; i < len(times); i++ {
		stringTimes = stringTimes + "," + strconv.FormatInt((times[i]-times[i-1])/1000000, 10)
	}
	jww.INFO.Print("Ping time between n nodes; n -> 1, 1 -> 2, " +
		"..., n-1 -> n (ms): " + stringTimes)
}
