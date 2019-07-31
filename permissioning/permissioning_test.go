////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"bytes"
	"fmt"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/services"
	"math/rand"
	"strings"
	"testing"
)

var nodeId *id.Node
var permComms *registration.RegistrationComms

// Dummy implementation of permissioning server -----------------------
type Implementation struct{}

func (i *Implementation) RegisterUser(registrationCode string, Y, P, Q,
	G []byte) (hash, R, S []byte, err error) {
	return nil, nil, nil, nil
}
func (i *Implementation) RegisterNode(ID []byte,
	NodeTLSCert, GatewayTLSCert, RegistrationCode, Addr string) error {
	Addr = strings.Replace(Addr, "127.0.0.1", "0.0.0.0", -1)
	_ = permComms.ConnectToNode(nodeId, Addr, nil)
	nodeTop := make([]*pb.NodeInfo, 0)
	nodeTop = append(nodeTop, &pb.NodeInfo{
		Id:        nodeId.Bytes(),
		Index:     0,
		IpAddress: Addr,
		TlsCert:   "",
	})
	nwTop := &pb.NodeTopology{
		Topology: nodeTop,
	}
	_ = permComms.SendNodeTopology(nodeId, nwTop)
	return nil
}

// --------------------------------------------------------------------

func TestRegisterNode(t *testing.T) {
	// Initialize permissioning server
	pAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	handler := registration.Handler(&Implementation{})
	permComms = registration.StartRegistrationServer(pAddr, handler, nil, nil)

	// Initialize definition
	nodeId = id.NewNodeFromUInt(uint64(0), t)
	addr := fmt.Sprintf("0.0.0.0:%d", 6000+rand.Intn(1000))
	def := &server.Definition{
		Flags:         server.Flags{},
		ID:            nodeId,
		DsaPublicKey:  nil,
		DsaPrivateKey: nil,
		TlsCert:       make([]byte, 0),
		TlsKey:        nil,
		Address:       addr,
		LogPath:       "",
		MetricLogPath: "",
		Gateway: server.GW{
			TlsCert: nil,
		},
		UserRegistry:    nil,
		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		BatchSize:       0,
		CmixGroup:       nil,
		E2EGroup:        nil,
		Topology:        nil,
		Nodes:           make([]server.Node, 1),
		Permissioning: server.Perm{
			TlsCert:          nil,
			RegistrationCode: "",
			Address:          pAddr,
		},
	}

	// Register the node
	RegisterNode(def)

	n := def.Nodes
	if len(n) < 1 {

	}
	if bytes.Compare(n[0].ID.Bytes(), nodeId.Bytes()) != 0 {

	}
	if n[0].Address != addr {

	}
	if n[0].TlsCert == nil {

	}
}
