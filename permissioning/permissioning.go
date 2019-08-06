////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains interactions with the Node Permissioning Server

package permissioning

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"net"
	"time"
)

// Stringer object for Permissioning connection ID
type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *server.Definition) {

	// Channel for signaling completion of Node registration
	toplogyCh := make(chan *pb.NodeTopology)
	gatewayCertsCh := make(chan *pb.NodeInfo)
	gatewayReadyCh := make(chan struct{}, 1)

	// Assemble the Comms callback interface
	impl := node.NewImplementation()
	impl.Functions.DownloadTopology = func(info *node.MessageInfo,
		topology *pb.NodeTopology) {
		// Signal completion of Node registration
		toplogyCh <- topology
	}

	impl.Functions.GetSignedCert = func(ping *pb.Ping) (*pb.SignedCerts, error) {
		certs := pb.SignedCerts{}
		select {
		case nodeInfo := <-gatewayCertsCh:
			certs.GatewayCertPEM = nodeInfo.GatewayTlsCert
			certs.ServerCertPEM = nodeInfo.ServerTlsCert
			gatewayReadyCh <- struct{}{}
		case <-time.After(1 * time.Second):
		}
		return &certs, nil
	}

	// Start Node communication server
	network := node.StartNode(def.Address, impl, def.TlsCert, def.TlsKey)
	permissioningId := ConnAddr("Permissioning")

	// Connect to the Permissioning Server
	err := network.ConnectToRegistration(permissioningId,
		def.Permissioning.Address, def.Permissioning.TlsCert)
	if err != nil {
		jww.FATAL.Panicf("Unable to initiate Node registration: %+v",
			errors.New(err.Error()))
	}

	// Connect to the Gateway
	err = network.ConnectToGateway(def.ID.NewGateway(),
		def.Gateway.Address, def.Gateway.TlsCert)
	if err != nil {
		jww.FATAL.Panicf("Unable to initiate Node registration: %+v",
			errors.New(err.Error()))
	}

	// Attempt Node registration
	_, port, err := net.SplitHostPort(def.Address)
	if err != nil {
		jww.FATAL.Panicf("Unable to obtain port from address: %+v",
			errors.New(err.Error()))
	}
	err = network.SendNodeRegistration(permissioningId,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			NodeCsr:          string(def.TlsCert),
			GatewayCsr:       string(def.Gateway.TlsCert),
			RegistrationCode: def.Permissioning.RegistrationCode,
			Port:             port,
		})
	if err != nil {
		jww.FATAL.Panicf("Unable to send Node registration: %+v",
			errors.New(err.Error()))
	}

	// Wait for Node registration to complete
	topology := <-toplogyCh

	//send certs to the gateway
	index := -1
	for i, n := range topology.Topology {
		// Update Cert for this Node
		if bytes.Compare(n.Id, def.ID.Bytes()) == 0 {
			index = i
		}

	}
	gatewayCertsCh <- topology.Topology[index]

	//Wait for gateway to be ready
	<-gatewayReadyCh
	time.Sleep(1 * time.Second)

	// Shut down the Comms server
	network.Shutdown()

	// Integrate the topology with the Definition
	def.Nodes = make([]server.Node, len(topology.Topology))
	for _, n := range topology.Topology {

		// Build Node information
		def.Nodes[n.Index] = server.Node{
			ID:      id.NewNodeFromBytes(n.Id),
			TlsCert: []byte(n.ServerTlsCert),
			Address: n.IpAddress,
		}

		// Update Cert for this Node
		if bytes.Compare(n.Id, def.ID.Bytes()) == 0 {
			def.TlsCert = []byte(n.ServerTlsCert)
			def.Gateway.TlsCert = []byte(n.GatewayTlsCert)
		}

	}

}
