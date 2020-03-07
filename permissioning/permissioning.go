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
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/server"
	"strconv"
	"strings"
	"sync"
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

// Poll handles the server requesting the ndf from permissioning
// it also holds the callback which handles gateway requesting an ndf from its server
func Poll(permHost *connect.Host, instance *server.Instance) error {

	// Initialize variable useful for polling
	errChan := make(chan error) // Used to check errors
	done := make(chan struct{}) // Used to signal that polling has completed once
	var once sync.Once          // Used to only send to above channel once
	sendDoneSignal := func() {  //  for the continuous loop
		done <- struct{}{}
	}

	// Routinely poll permissioning for state updates
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		// Continuously poll permissioning for information
		for {
			select {
			case <-ticker.C:
				permResponse := RetrieveState(permHost, instance, errChan)
				UpdateInternalState(permResponse, instance, errChan)

			}
			// Only send once to avoid a memory leak
			once.Do(sendDoneSignal)
		}
	}()

	// Wait for the go function to complete once
	<-done

	//See if the polling has returned errors
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}

	}

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	jww.INFO.Printf("Successfully obtained NDF!")
	return errs

}

// RetrieveState polls the permissioning server for updates and
func RetrieveState(permHost *connect.Host,
	instance *server.Instance, errorChan chan error) *pb.PermissionPollResponse {
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
	permissioningResponse, err := instance.GetNetwork().SendPoll(permHost, pollMsg)
	if err != nil {
		errorChan <- errors.Errorf("Issue polling permissioning: %+v", err)
	}

	return permissioningResponse
}

// UpdateState processes the polling response from permissioning, installing any changes if needed
func UpdateInternalState(permissioningResponse *pb.PermissionPollResponse, instance *server.Instance, errorChan chan error) {

	// Parse the response for updates
	newUpdates := permissioningResponse.Updates

	// Update round info
	for _, roundInfo := range newUpdates {
		err := instance.GetConsensus().RoundUpdate(roundInfo)
		if err != nil {
			errorChan <- errors.Errorf("Unable to update for round %+v: %+v", roundInfo.ID, err)
		}
	}

	// Update the full ndf
	err := instance.GetConsensus().UpdateFullNdf(permissioningResponse.FullNDF)
	if err != nil {
		errorChan <- err
	}

	// Update the partial ndf
	err = instance.GetConsensus().UpdatePartialNdf(permissioningResponse.PartialNDF)
	if err != nil {
		errorChan <- err
	}

	// Update the nodes in the network.Instance with the new ndf
	err = instance.GetConsensus().UpdateNodeConnections()
	if err != nil {
		errorChan <- err
	}

	// Update the gateways in the network.Instance with the new ndf
	err = instance.GetConsensus().UpdateGatewayConnections()
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

// todo: delete function??

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
