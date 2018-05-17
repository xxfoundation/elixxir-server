package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"sync"
)

var IsLastNode bool

var nodeId uint64
var setNodeIdOnce sync.Once

func SetNodeID(newNodeID uint64) {
	setNodeIdOnce.Do(func() {
		nodeId = newNodeID
		jww.DEBUG.Printf("Node ID: %v", nodeId)
	})
}

func GetNodeID() uint64 {
	setNodeIdOnce.Do(func() {
		nodeId = uint64(viper.GetInt("nodeID"))
		jww.WARN.Printf("Set node ID from Viper. Node ID: %v", nodeId)
	})
	return nodeId
}
