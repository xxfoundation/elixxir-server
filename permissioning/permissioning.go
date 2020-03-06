////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains interactions with the Node Permissioning Server

package permissioning

import (
	"bytes"
	"github.com/jasonlvhit/gocron"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/server"
	"strconv"
	"strings"
	"time"
)

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *server.Definition, network *node.Comms, permHost *connect.Host) error {
	// We don't check validity here, because the registration server should.
	gw := strings.Split(def.Gateway.Address, ":")
	gwPort, _ := strconv.ParseUint(gw[1], 10, 32)
	node := strings.Split(def.Address, ":")
	nodePort, _ := strconv.ParseUint(node[1], 10, 32)
	// Attempt Node registration
	err := network.SendNodeRegistration(permHost,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			ServerTlsCert:    string(def.TlsCert),
			GatewayTlsCert:   string(def.Gateway.TlsCert),
			GatewayAddress:   gw[0],
			GatewayPort:      uint32(gwPort),
			ServerAddress:    node[0],
			ServerPort:       uint32(nodePort),
			RegistrationCode: def.Permissioning.RegistrationCode,
		})
	if err != nil {
		return errors.Errorf("Unable to send Node registration: %+v", err)
	}

	return nil
}

// PollNdf handles the server requesting the ndf from permissioning
// it also holds the callback which handles gateway requesting an ndf from its server
func PollNdf(def *server.Definition, network *node.Comms, permHost *connect.Host, instance *server.Instance) error {

	// Keep polling until there is a response (ie no error)
	errChan := make(chan error)
	done := make(chan struct{})
	go func() {
		for {
			RetrieveNdf(permHost, network, instance, errChan)
			time.Sleep(500 * time.Millisecond)
			done <- struct{}{}
		}
	}()

	<-done

	if len(errChan) != 0 {

	}

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	jww.INFO.Printf("Successfully obtained NDF!")
	return nil

}

// todo: determine if this belongs here or comms or something
//  after first time it autoruns?
//  if you handle connectivity b4
//  it should handle
//  if they groups don't match, error
func RetrieveNdf(permHost *connect.Host, network *node.Comms, instance *server.Instance, errorChan chan error) {
	// Get the ndf hashes for partial and full ndf
	var fullNdfHash, partialNdfHash []byte
	if instance.GetConsensus().GetFullNdf() != nil {
		fullNdfHash = instance.GetConsensus().GetFullNdf().GetHash()
	}
	if instance.GetConsensus().GetPartialNdf() != nil {
		partialNdfHash = instance.GetConsensus().GetPartialNdf().GetHash()
	}

	// Get the update id and activity of the state machine
	lastUpdateId := instance.GetConsensus().GetLastUpdateID()
	activity := instance.GetStateMachine().Get()

	// Construct a message for permissioning with above information
	pollMsg := &pb.PermissioningPoll{
		Full:       &pb.NDFHash{Hash: fullNdfHash},
		Partial:    &pb.NDFHash{Hash: partialNdfHash},
		LastUpdate: uint64(lastUpdateId),
		Activity:   uint32(activity),
	}

	// Send the message to permissioning
	permissioningResponse, err := network.SendPoll(permHost, pollMsg)
	if err != nil {
		errorChan <- errors.Errorf("Issue polling permissioning: %+v", err)
	}

	// Parse the response for updates
	newUpdates := permissioningResponse.Updates

	// update instance logic...
	// todo: figure out if this should be outside of this function or inside, decide how go-func shit should work
	//  and figure out any race conditions
	for _, roundInfo := range newUpdates {
		err = instance.GetConsensus().RoundUpdate(roundInfo)
		if err != nil {
			errorChan <- errors.Errorf("Unable to update for round %+v: %+v", roundInfo.ID, err)
		}
	}

	// Update the full ndf
	err = instance.GetConsensus().UpdateFullNdf(permissioningResponse.FullNDF)
	if err != nil {
		errorChan <- err
	}

	// Update the full ndf
	err = instance.GetConsensus().UpdatePartialNdf(permissioningResponse.PartialNDF)
	if err != nil {
		errorChan <- err
	}

	// Update the full ndf
	err = initializeHosts(instance.GetConsensus().GetFullNdf().Get(), network, instance.GetID().Bytes())
	if err != nil {
		errorChan <- err
	}

}

// InstallNdf parses the ndf for necessary information and returns that
func InstallNdf(def *server.Definition, newNdf *ndf.NetworkDefinition) (string, string, error) {

	jww.INFO.Println("Installing NDF now...")

	index, err := findOurNode(def.ID.Bytes(), newNdf.Nodes)
	if err != nil {
		return "", "", err
	}

	//Fixme: at some point soon we will not be able to assume the node & corresponding gateway share the same index
	// will need to add logic to find the corresponding gateway..
	return newNdf.Nodes[index].TlsCertificate, newNdf.Gateways[index].TlsCertificate, nil
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
func initializeHosts(def *ndf.NetworkDefinition, network *node.Comms, ourId []byte) error {
	// Find this server's place in the ndf
	myIndex, err := findOurNode(ourId, def.Nodes)
	if err != nil {
		return err
	}

	// Add hosts for nodes
	for i, host := range def.Nodes {
		_, err := network.AddHost(id.NewNodeFromBytes(host.ID).String(),
			host.Address, []byte(host.TlsCertificate), false, true)
		if err != nil {
			return errors.Errorf("Unable to add host for gateway %d at %+v", i, host.Address)
		}
	}

	// Add host for the relevant gateway
	gateway := def.Gateways[myIndex]
	_, err = network.AddHost(id.NewNodeFromBytes(def.Nodes[myIndex].ID).NewGateway().String(),
		gateway.Address, []byte(gateway.TlsCertificate), false, true)
	if err != nil {
		return errors.Errorf("Unable to add host for gateway %s at %+v", network.String(), gateway.Address)
	}
	return nil
}
