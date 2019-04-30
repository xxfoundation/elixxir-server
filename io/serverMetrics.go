////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/globals"

	//linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/node"
	"runtime"
	"strconv"
)

// Records current time and sends all recorded times to next node
func ServerMetrics(msg *pb.ServerMetricsMessage) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsage := append(msg.MemUsage, uint32((m.Alloc+m.HeapAlloc)/1024/1024))
	threadUsage := append(msg.ThreadUsage, uint32(runtime.NumGoroutine()))
	cpuUsage := append(msg.CpuUsage, uint32(0))
	upSince := append(msg.UpSince, TimeUp)
	if !globals.IsLastNode {
		node.SendServerMetrics(Servers[len(upSince)],
			&pb.ServerMetricsMessage{
				MemUsage:    memUsage,
				ThreadUsage: threadUsage,
				CpuUsage:    cpuUsage,
				UpSince:     upSince,
			})
	} else {
		LogServerMetrics(memUsage, threadUsage, cpuUsage, upSince)
	}
}

// Initiates a roundtrip ping starting at last node
func GetServerMetrics(servers []string) {
	memUsage := make([]uint32, 0)

	threadUsage := make([]uint32, 0)

	// TODO Need to add CPU metrics
	cpuUsage := make([]uint32, 0)
	/*if runtime.GOOS == "linux" {
		linuxproc
	}*/

	upSince := make([]int64, 0)

	// if only one node then just return the metrics for that node
	if len(servers) < 2 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memUsage = append(memUsage, uint32((m.Alloc+m.HeapAlloc)/1024/1024))
		threadUsage = append(threadUsage, uint32(runtime.NumGoroutine()))
		cpuUsage = append(cpuUsage, uint32(0))
		upSince = append(upSince, TimeUp)
		LogServerMetrics(memUsage, threadUsage, cpuUsage, upSince)
		// else send to first node
	} else {
		node.SendServerMetrics(servers[0],
			&pb.ServerMetricsMessage{
				MemUsage:    memUsage,
				ThreadUsage: threadUsage,
				CpuUsage:    cpuUsage,
				UpSince:     upSince,
			})

	}
}

// Logs the results of the roundtrip ping in milliseconds between nodes
func LogServerMetrics(memUsage []uint32, threadUsage []uint32,
	cpuUsage []uint32, upSince []int64) {
	metrics := "Server metric dump: "
	if len(viper.GetStringSlice("servers")) == len(memUsage) {
		for i := 0; i < len(memUsage); i++ {
			metrics = metrics + viper.GetStringSlice("servers")[i] + "," +
				strconv.FormatUint(uint64(memUsage[i]), 10) + "," +
				strconv.FormatUint(uint64(cpuUsage[i]), 10) + "," +
				strconv.FormatUint(uint64(threadUsage[i]), 10) + "," +
				strconv.FormatInt(upSince[i], 10) + ";"
		}
	}
	jww.INFO.Print(metrics)
}
