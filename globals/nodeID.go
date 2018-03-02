package globals

import (
	"github.com/spf13/viper"
	"math"
	jww "github.com/spf13/jwalterweatherman"
)

var nodeId = uint64(math.MaxUint64)

func NodeID(serverIdx uint64) uint64 {
	if nodeId == math.MaxUint64 {
		jww.DEBUG.Printf("Getting node ID")
		if !viper.IsSet("nodeID") {
			// set node ID to server index as a backup
			nodeId = serverIdx
		}
		nodeId := viper.GetInt("nodeID")
		jww.DEBUG.Printf("Node ID: %v", nodeId)
	}
	return nodeId
}
