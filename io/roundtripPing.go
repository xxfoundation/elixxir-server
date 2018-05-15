package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/server/globals"
	"time"
)

// Blocks until all given servers respond
func RoundtripPing(msg *pb.TimePing) {
	if !globals.IsLastNode {
		times := make([]int64,0)
		for i := range msg.Times {
			times = append(times, msg.Times[i])
		}
		times = append(times, time.Now().UnixNano())
		node.SendRoundtripPing(Servers[len(times)], &pb.TimePing{times})
	} else {
		times := make([]int64,0)
		for i := range msg.Times {
			times = append(times, msg.Times[i])
		}
		times = append(times, time.Now().UnixNano())
		// TODO Do something useful with this slice of times
	}

}

func GetRoundtripPing(servers []string) {
	times := make([]int64,0)
	times = append(times, time.Now().UnixNano())
	if !globals.IsLastNode {
		node.SendRoundtripPing(servers[1], &pb.TimePing{times})
	}
}
