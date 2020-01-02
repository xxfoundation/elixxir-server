////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains interactions with the Node Permissioning Server

package permissioning

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/server"
	"time"
)

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *server.Definition, network *node.Comms, permHost *connect.Host) error {
	// Attempt Node registration
	err := network.SendNodeRegistration(permHost,
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

	return nil
}

//PollNdf handles the server requesting the ndf from permissioning
// it also holds the callback which handles gateway requesting an ndf from its server
func PollNdf(def *server.Definition, network *node.Comms,
	gatewayNdfChan chan *pb.GatewayNdf, gatewayReadyCh chan struct{}, permHost *connect.Host) (*ndf.NetworkDefinition, error) {
	// Keep polling until there is a response (ie no error)
	var response *pb.NDF
	var err error

	jww.INFO.Printf("Beginning polling NDF...")
	for response == nil {
		jww.DEBUG.Printf("Polling for Ndf...")
		response, err = network.RequestNdf(permHost,
			&pb.NDFHash{Hash: make([]byte, 0)})
		if err != nil {
			return nil, errors.Errorf("Unable to poll for Ndf: %+v", err)
		}
	}

	// Decode the ndf into an object
	newNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		errMsg := errors.Errorf("Unable to parse Ndf: %v", err)
		return nil, errMsg
	}
	// Find this server's place in the ndf
	index, err := findOurNode(def.ID.Bytes(), newNdf.Nodes)
	if err != nil {
		return nil, err
	}

	err = initializeHosts(newNdf, network, index)

	//Prepare the ndf for gateway transmission
	ndfData, err := json.Marshal(newNdf)
	if err != nil {
		return nil, err
	}

	//Send the certs to the gateway
	gatewayNdfChan <- &pb.GatewayNdf{
		Id:  newNdf.Nodes[index].ID,
		Ndf: &pb.NDF{Ndf: ndfData},
	}

	//Wait for gateway to be ready
	<-gatewayReadyCh

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

// initializeHosts adds host objects for all relevant connections in the NDF
func initializeHosts(def *ndf.NetworkDefinition, network *node.Comms, myIndex int) error {
	// Add hosts for nodes
	for i, host := range def.Nodes {
		if i == myIndex {
			continue
		}
		_, err := network.AddHost(id.NewNodeFromBytes(host.ID).String(),
			host.Address, []byte(host.TlsCertificate), false, true)
		if err != nil {
			return errors.Errorf("Unable to add host for gateway %d at %+v", i, host.Address)
		}
	}

	// Add host for the relevant gateway
	gateway := def.Gateways[myIndex]
	_, err := network.AddHost(id.NewNodeFromBytes(def.Nodes[myIndex].ID).String(), gateway.Address,
		[]byte(gateway.TlsCertificate), false, true)
	if err != nil {
		return errors.Errorf("Unable to add host for gateway %s at %+v", network.String(), gateway.Address)
	}
	return nil
}
