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
	"gitlab.com/elixxir/primitives/ndf"
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
func RegisterNode(def *server.Definition) error {
	// Assemble the Comms callback interface
	impl := node.NewImplementation()

	// Start Node communication server
	network := node.StartNode(def.Address, impl, def.TlsCert, def.TlsKey)
	// Connect to the Permissioning Server

	permHost, err := network.AddHost(id.PERMISSIONING, def.Permissioning.Address, def.Permissioning.TlsCert, true)
	if err != nil {
		errMsg := errors.Errorf("Unable to connect to registration server: %+v", err)
		return errMsg
	}

	// Attempt Node registration
	_, _, err = net.SplitHostPort(def.Address)
	if err != nil {
		errMsg := errors.Errorf("Unable to obtain port from address: %+v", err)
		return errMsg
	}

	err = network.SendNodeRegistration(permHost,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			ServerTlsCert:    string(def.TlsCert),
			GatewayTlsCert:   string(def.Gateway.TlsCert),
			GatewayAddress:   def.Gateway.Address,
			RegistrationCode: def.Permissioning.RegistrationCode,
		})
	if err != nil {
		errMsg := errors.Errorf("Unable to send Node registration: %+v", err)
		return errMsg
	}

	return nil
}

//PollNdf polls permissioning for an ndf
func PollNdf(def *server.Definition) (*ndf.NetworkDefinition, error) {

	// Assemble the Comms callback interface
	impl := node.NewImplementation()

	// Start Node communication server
	network := node.StartNode(def.Address, impl, def.TlsCert, def.TlsKey)
	// Connect to the Permissioning Server
	permHost, err := network.AddHost(id.PERMISSIONING, def.Permissioning.Address, def.Permissioning.TlsCert, true)
	if err != nil {
		errMsg := errors.Errorf("Unable to connect to registration server: %+v", err)
		return nil, errMsg
	}

	jww.INFO.Printf("Beginning polling NDF...")
	// Keep polling until there is a response (ie no error)
	var response *pb.NDF
	for response == nil || err != nil {
		response, err = network.RequestNdf(permHost, nil)
		//When permissioning has not timed out, stop polling
	}

	newNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		errMsg := errors.Errorf("Unable to parse ndf: %v", err)
		return nil, errMsg
	}
	// Shut down the Comms server
	network.Shutdown()

	jww.INFO.Printf("Successfully obtained NDF!")

	return newNdf, nil

}

//InstallNdf parses the ndf for useful information and handles gateway certs comm
func InstallNdf(def *server.Definition, newNdf *ndf.NetworkDefinition) ([]server.Node, []*id.Node,
	string, string, error) {
	// Channel for signaling completion of Node registration
	gatewayNdfChan := make(chan *pb.GatewayNdf)
	gatewayReadyCh := make(chan struct{}, 1)
	// Assemble the Comms callback interface
	impl := node.NewImplementation()

	// Assemble the Comms callback interface
	impl.Functions.PollNdf = func(ping *pb.Ping) (*pb.GatewayNdf, error) {
		gwNdf := pb.GatewayNdf{}
		select {
		case gwNdf = <-gatewayNdfChan:
			gatewayReadyCh <- struct{}{}
		case <-time.After(1 * time.Second):
		}
		return &gwNdf, nil
	}

	jww.INFO.Println("Installing NDF now...")
	//Find this node's place in the newNDF
	index := -1
	for i, newNode := range newNdf.Nodes {
		//Use that index bookkeeping purposes when later parsing ndf
		if bytes.Compare(newNode.ID, def.ID.Bytes()) == 0 {
			index = i
		}
	}

	//Send the certs to the gateway
	gatewayNdfChan <- &pb.GatewayNdf{
		Id:  newNdf.Nodes[index].ID,
		Ndf: &pb.NDF{Ndf: newNdf.Serialize()},
	}

	//Wait for gateway to be ready
	<-gatewayReadyCh
	time.Sleep(1 * time.Second)

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	// Integrate the topology with the Definition
	nodes := make([]server.Node, len(newNdf.Nodes))
	nodeIds := make([]*id.Node, len(newNdf.Nodes))
	for i, newNode := range newNdf.Nodes {
		// Build Node information
		jww.INFO.Printf("Assembling node topology: %+v", newNode)
		nodes[i] = server.Node{
			ID:      id.NewNodeFromBytes(newNode.ID),
			TlsCert: []byte(newNode.TlsCertificate),
			Address: newNode.Address,
		}
		nodeIds[i] = id.NewNodeFromBytes(newNode.ID)
	}

	//Fixme: at some point soon we will not be able to assume the node & corresponding gateway share the same index
	// will need to add logic to find the corresponding gateway..
	return nodes, nodeIds, newNdf.Nodes[index].TlsCertificate, newNdf.Gateways[index].TlsCertificate, nil
}
