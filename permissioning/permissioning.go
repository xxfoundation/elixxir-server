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
	"gitlab.com/elixxir/comms/connect"
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
func RegisterNode(def *server.Definition) ([]server.Node, []*id.Node, string,
	string) {

	// Channel for signaling completion of Node registration
	toplogyCh := make(chan *pb.NodeTopology)
	gatewayCertsCh := make(chan *pb.NodeInfo)
	gatewayReadyCh := make(chan struct{}, 1)

	// Assemble the Comms callback interface
	impl := node.NewImplementation()
	impl.Functions.DownloadTopology = func(info *node.MessageInfo, topology *pb.NodeTopology) {
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
	// Connect to the Permissioning Server
	permHost, err := connect.NewHost(def.Permissioning.Address, def.Permissioning.TlsCert, true)
	if err != nil {
		jww.FATAL.Panicf("Unable to connect to registration server: %+v", errors.New(err.Error()))
	}

	network.AddHost(id.PERMISSIONING, permHost)

	// Attempt Node registration
	_, port, err := net.SplitHostPort(def.Address)
	if err != nil {
		jww.FATAL.Panicf("Unable to obtain port from address: %+v",
			errors.New(err.Error()))
	}

	err = network.SendNodeRegistration(permHost,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			ServerTlsCert:    string(def.TlsCert),
			GatewayTlsCert:   string(def.Gateway.TlsCert),
			GatewayAddress:   def.Gateway.Address,
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

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	// Integrate the topology with the Definition
	nodes := make([]server.Node, len(topology.Topology))
	nodeIds := make([]*id.Node, len(topology.Topology))
	for _, n := range topology.Topology {
		// Build Node information
		jww.INFO.Printf("Assembling node topology: %+v", n)
		nodes[n.Index] = server.Node{
			ID:      id.NewNodeFromBytes(n.Id),
			TlsCert: []byte(n.ServerTlsCert),
			Address: n.ServerAddress,
		}
		nodeIds[n.Index] = id.NewNodeFromBytes(n.Id)
	}

	return nodes, nodeIds, topology.Topology[index].ServerTlsCert,
		topology.Topology[index].GatewayTlsCert

}
