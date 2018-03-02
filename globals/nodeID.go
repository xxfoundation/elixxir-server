package globals

import (
	"github.com/spf13/viper"
	"math"
	jww "github.com/spf13/jwalterweatherman"
)

var nodeId = uint64(math.MaxUint64)

func NodeID() uint64 {
	if nodeId == math.MaxUint64 {
		jww.DEBUG.Printf("Getting node ID")
		if !viper.IsSet("nodeID") {
			panic("Node ID wasn't set")
		}
		nodeId := viper.GetInt("nodeID")
		jww.DEBUG.Printf("Node ID: %v", nodeId)
	}
	return nodeId
}
