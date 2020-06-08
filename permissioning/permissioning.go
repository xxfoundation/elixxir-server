////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains interactions with the Node Permissioning Server

package permissioning

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/internal"
	"strconv"
	"strings"
	"time"
)

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *internal.Definition, network *node.Comms, permHost *connect.Host) error {
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
			RegistrationCode: def.RegistrationCode,
		})
	if err != nil {
		return errors.Errorf("Unable to send Node registration: %+v", err)
	}

	return nil
}

// Poll is used to retrieve updated state information from permissioning
//  and update our internal state accordingly
func Poll(instance *internal.Instance) error {

	// Fetch the host information from the network
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		return errors.New("Could not get permissioning host")
	}

	var reportedActivity current.Activity
	select {
	case reportedActivity = <-instance.GetStateMachine().GetBuffer():
	default:
		reportedActivity = instance.GetStateMachine().Get()
	}

	// Once done and in a completed state, manually switch back into waiting
	if reportedActivity == current.COMPLETED {
		ok, err := instance.GetStateMachine().Update(current.WAITING)
		if err != nil || !ok {
			return errors.Errorf("Could not transition to WAITING state: %v", err)
		}
	}

	// Ping permissioning for updated information
	permResponse, err := PollPermissioning(permHost, instance, reportedActivity)
	if err != nil {
		return err
	}

	//updates the NDF with changes
	err = UpdateNDf(permResponse, instance)
	if err != nil {
		return errors.WithMessage(err, "Failed to update the NDFs")
	}

	// Update the internal state of rounds and the state machine
	err = UpdateRounds(permResponse, instance)
	return err
}

// PollPermissioning polls the permissioning server for updates
func PollPermissioning(permHost *connect.Host, instance *internal.Instance, reportedActivity current.Activity) (*pb.PermissionPollResponse, error) {
	var fullNdfHash, partialNdfHash []byte

	// Get the ndf hashes for the full ndf if available
	if instance.GetConsensus().GetFullNdf() != nil {
		fullNdfHash = instance.GetConsensus().GetFullNdf().GetHash()
	}

	// Get the ndf hashes for the partial ndf if available
	if instance.GetConsensus().GetPartialNdf() != nil {
		partialNdfHash = instance.GetConsensus().GetPartialNdf().GetHash()
	}

	// Get the update id and activity of the state machine
	lastUpdateId := instance.GetConsensus().GetLastUpdateID()

	// The ring buffer returns negative none but message type doesn't support signed numbers
	// fixme: maybe make proto have signed ints
	if lastUpdateId == -1 {
		lastUpdateId = 0
	}

	port, err := strconv.Atoi(strings.Split(instance.GetNetwork().ListeningAddr, ":")[1])
	if err != nil {
		jww.ERROR.Printf("Could not get port number out of server's address. " +
			"Likely this is because the address is an IPv6 address")
		return nil, err
	}

	gatewayAddr, gatewayVer := instance.GetGatewayData()

	// Construct a message for permissioning with above information
	pollMsg := &pb.PermissioningPoll{
		Full:       &pb.NDFHash{Hash: fullNdfHash},
		Partial:    &pb.NDFHash{Hash: partialNdfHash},
		LastUpdate: uint64(lastUpdateId),
		Activity:   uint32(reportedActivity),

		GatewayVersion: gatewayVer,
		GatewayAddress: gatewayAddr,

		ServerPort:    uint32(port),
		ServerVersion: instance.GetServerVersion(),
	}

	if instance.GetRecoveredError() != nil && instance.GetStateMachine().Get() == current.ERROR {
		pollMsg.Error = instance.GetRecoveredError()
		instance.ClearRecoveredError()
		go func() {
			ok, err := instance.GetStateMachine().Update(current.WAITING)
			if err != nil || !ok {
				jww.FATAL.Panicf("Could not move to waiting state")
			}
		}()
	}

	// Send the message to permissioning
	permissioningResponse, err := instance.GetNetwork().SendPoll(permHost, pollMsg)
	if err != nil {
		return nil, errors.Errorf("Issue polling permissioning: %+v", err)
	}

	return permissioningResponse, err
}

// Processes the polling response from permissioning for round updates,
// installing any round changes if needed. It also parsed the message and
// determines where to transition given context
func UpdateRounds(permissioningResponse *pb.PermissionPollResponse, instance *internal.Instance) error {

	// Parse the response for updates
	newUpdates := permissioningResponse.Updates
	// Parse the round info updates if they exist
	for _, roundInfo := range newUpdates {
		// Add the new information to the network instance
		err := instance.GetConsensus().RoundUpdate(roundInfo)
		if err != nil {
			return errors.Errorf("Unable to update for round %+v: %+v", roundInfo.ID, err)
		}

		// Extract topology from RoundInfo
		newNodeList, err := id.NewIDListFromBytes(roundInfo.Topology)
		if err != nil {
			return errors.Errorf("Unable to convert topology into a node list: %+v", err)
		}

		// fixme: this panics on error, external comm should not be able to crash server
		newTopology := connect.NewCircuit(newNodeList)

		// Check if our node is in this round
		if index := newTopology.GetNodeLocation(instance.GetID()); index != -1 {
			// Depending on the state in the roundInfo
			switch states.Round(roundInfo.State) {
			case states.PENDING:
				// Don't do anything
			case states.PRECOMPUTING: // Prepare for precomputing state

				// Standby until in WAITING state to ensure a valid transition into precomputing
				curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.WAITING)
				if curActivity != current.WAITING || err != nil {
					return errors.Errorf("Cannot start precomputing when not in waiting state: %+v", err)
				}

				// Send info to round queue
				err = instance.GetCreateRoundQueue().Send(roundInfo)
				if err != nil {
					return errors.Errorf("Unable to send to CreateRoundQueue: %+v", err)
				}

				// Begin PRECOMPUTING state
				ok, err := instance.GetStateMachine().Update(current.PRECOMPUTING)
				if !ok || err != nil {
					return errors.Errorf("Cannot move to precomputing state: %+v", err)
				}
			case states.STANDBY:
				// Don't do anything

			case states.QUEUED: // Prepare for realtime state
				// Wait until in STANDBY to ensure a valid transition into precomputing
				curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.STANDBY)
				if curActivity != current.STANDBY || err != nil {
					return errors.Errorf("Cannot start realtime when not in standby state: %+v", err)
				}

				// Send info to the realtime round queue
				err = instance.GetRealtimeRoundQueue().Send(roundInfo)
				if err != nil {
					return errors.Errorf("Unable to send to RealtimeRoundQueue: %+v", err)
				}

				// Wait until the permissioning-instructed time to begin REALTIME
				go func() {
					// Get the realtime start time
					duration := time.Unix(0, int64(roundInfo.Timestamps[states.QUEUED]))
					// Fixme: find way to calculate sleep length that doesn't lose time
					// If the timeDiff is positive, then we are not yet ready to start realtime.
					//  We then sleep for timeDiff time
					if timeDiff := time.Now().Sub(duration); timeDiff > 0 {
						jww.INFO.Printf("Sleeping for %+v ms for realtime start", timeDiff.Milliseconds())
						fmt.Println("sleeping for ", timeDiff.String())
						time.Sleep(timeDiff)
					}

					// Update to realtime when ready
					ok, err := instance.GetStateMachine().Update(current.REALTIME)
					if !ok || err != nil {
						jww.FATAL.Panicf("Cannot move to realtime state: %+v", err)
					}
				}()
			case states.REALTIME:
				// Don't do anything

			case states.COMPLETED:

			default:
				return errors.Errorf("Round in unknown state: %v", states.Round(roundInfo.State))

			}

		}
	}
	return nil
}

// Processes the polling response from permissioning for ndf updates,
// installing any ndf changes if needed and connecting to new nodes
func UpdateNDf(permissioningResponse *pb.PermissionPollResponse, instance *internal.Instance) error {
	if permissioningResponse.FullNDF != nil {
		jww.INFO.Printf("full ndf: %v", permissioningResponse.FullNDF)
		// Update the full ndf
		err := instance.GetConsensus().UpdateFullNdf(permissioningResponse.FullNDF)
		if err != nil {
			return errors.Errorf("Could not update full ndf: %+v", err)
		}
	}

	if permissioningResponse.PartialNDF != nil {
		jww.INFO.Printf("partial ndf: %v", permissioningResponse.PartialNDF)
		// Update the partial ndf
		err := instance.GetConsensus().UpdatePartialNdf(permissioningResponse.PartialNDF)
		if err != nil {
			return errors.Errorf("Could not update partial ndf: %+v", err)
		}
	}

	if permissioningResponse.PartialNDF != nil || permissioningResponse.FullNDF != nil {

		// Update the nodes in the network.Instance with the new ndf
		err := instance.GetConsensus().UpdateNodeConnections()
		if err != nil {
			return errors.Errorf("Could not update node connections: %+v", err)
		}
	}

	return nil

}

// InstallNdf parses the ndf for necessary information and returns that
func FindSelfInNdf(def *internal.Definition, newNdf *ndf.NetworkDefinition) (string, string, error) {

	jww.INFO.Println("Installing FullNDF now...")
	index, err := findOurNode(def.ID, newNdf.Nodes)
	if err != nil {
		return "", "", err
	}

	//Fixme: at some point soon we will not be able to assume the node & corresponding gateway share the same index
	// will need to add logic to find the corresponding gateway..
	return newNdf.Nodes[index].TlsCertificate, // it also holds the callback which handles gateway requesting an ndf from its server
		newNdf.Gateways[index].TlsCertificate, nil
}

//findOurNode is a helper function which finds our node's index in the ndf
// it returns the index of our node if found or an error if not found
func findOurNode(nodeId *id.ID, nodes []ndf.Node) (int, error) {
	//Find this node's place in the newNDF
	for i, newNode := range nodes {
		//Use that index bookkeeping purposes when later parsing ndf
		if bytes.Compare(newNode.ID, nodeId.Bytes()) == 0 {
			return i, nil
		}
	}
	return -1, errors.New("Failed to find node in ndf, maybe node registration failed?")

}
