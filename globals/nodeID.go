package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"math"
)

var IsLastNode bool

var nodeId = uint64(math.MaxUint64)

func NodeID(serverIdx uint64) uint64 {
	if nodeId == uint64(math.MaxUint64) {
		viperNodeId := uint64(viper.GetInt("nodeID"))

		// viper.IsSet() doesn't do what I want to do here, because binding it
		// as a persistent flag in cmd/root.go always sets IsSet() to true
		// for this flag. So, we check to see if the result from viper is 0
		// instead. See https://github.com/spf13/viper/pull/331
		if viperNodeId == 0 {
			// set node ID to server index as a backup
			nodeId = serverIdx
		} else {
			nodeId = viperNodeId
		}
		jww.DEBUG.Printf("Node ID: %v", nodeId)
	}
	return nodeId
}
