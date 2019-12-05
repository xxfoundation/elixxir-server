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

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *server.Definition) error {
	// Assemble the Comms callback interface
	impl := node.NewImplementation()

	// Start Node communication server
	network := node.StartNode(def.Address, impl, def.TlsCert, def.TlsKey)
	// Connect to the Permissioning Server
	permHost, err := network.AddHost(id.PERMISSIONING, def.Permissioning.Address, def.Permissioning.TlsCert, true)
	if err != nil {
		errMsg := errors.Errorf("Unable to create registration host: %+v", err)
		return errMsg
	}

	_, _, err = net.SplitHostPort(def.Address)
	if err != nil {
		errMsg := errors.Errorf("Unable to obtain port from address: %+v", err)
		return errMsg
	}

	// Attempt Node registration
	err = network.SendNodeRegistration(permHost,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			ServerTlsCert:    string(def.TlsCert),
			GatewayTlsCert:   string(def.Gateway.TlsCert),
			GatewayAddress:   def.Gateway.Address,
			RegistrationCode: def.Permissioning.RegistrationCode,
		})
	if err != nil {
		return errors.Errorf("Unable to send Node registration: %+v", err)
	}

	//Shutdown the temp network and return no error
	network.Shutdown()

	return nil
}

//PollNdf handles the server requesting the ndf from permissioning
// it also holds the callback which handles gateway requesting an ndf from its server
func PollNdf(def *server.Definition) (*ndf.NetworkDefinition, error) {
	// Channel for signaling completion of Node registration
	gatewayNdfChan := make(chan *pb.GatewayNdf)
	gatewayReadyCh := make(chan struct{}, 1)

	// Assemble the Comms callback interface
	impl := node.NewImplementation()

	// Assemble the Comms callback interface
	impl.Functions.PollNdf = func(ping *pb.Ping) (*pb.GatewayNdf, error) {
		var gwNdf *pb.GatewayNdf
		select {
		case gwNdf = <-gatewayNdfChan:
			jww.DEBUG.Println("Giving ndf to gateway")
			gatewayReadyCh <- struct{}{}
		case <-time.After(1 * time.Second):
		}
		return gwNdf, nil

	}
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
		response, err = network.RequestNdf(permHost, &pb.NDFHash{})
	}

	//Decode the ndf into an object
	newNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		errMsg := errors.Errorf("Unable to parse ndf: %v", err)
		return nil, errMsg
	}
	//Find this server's place in the ndf
	index, err := findOurNode(def.ID.Bytes(), newNdf.Nodes)
	if err != nil {
		return nil, err
	}

	//Send the certs to the gateway
	gatewayNdfChan <- &pb.GatewayNdf{
		Id:  newNdf.Nodes[index].ID,
		Ndf: &pb.NDF{Ndf: newNdf.Serialize()},
	}

	//Wait for gateway to be ready
	<-gatewayReadyCh

	// Shut down the Comms server and return the ndf
	network.Shutdown()

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	jww.INFO.Printf("Successfully obtained NDF!")
	return newNdf, nil

}

//InstallNdf parses the ndf for necessary information and returns that
func InstallNdf(def *server.Definition, newNdf *ndf.NetworkDefinition) ([]server.Node, []*id.Node,
	string, string, error) {

	jww.INFO.Println("Installing NDF now...")

	index, err := findOurNode(def.ID.Bytes(), newNdf.Nodes)
	if err != nil {
		return nil, nil, "", "", err
	}

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

//findOurNode is a helper function which finds our node's index in the ndf
// it returns the index of our node if found or an error if not found
func findOurNode(nodeId []byte, nodes []ndf.Node) (int, error) {
	//Find this node's place in the newNDF
	for i, newNode := range nodes {
		//Use that index bookkeeping purposes when later parsing ndf
		if bytes.Compare(newNode.ID, nodeId) == 0 {
			return i, nil
		}
	}
	return -1, errors.New("Failed to find node in ndf, maybe node registration failed?")

}
