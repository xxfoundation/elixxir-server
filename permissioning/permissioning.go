////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Contains interactions with the Node Permissioning Server

package permissioning

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Cache path constants for NDF caching
const (
	fullNdfCachePath    = "/opt/xxnetwork/cache/full_ndf.json"
	partialNdfCachePath = "/opt/xxnetwork/cache/partial_ndf.json"
)

// GetFullNdfCachePath returns the full NDF cache file path
func GetFullNdfCachePath() string {
	return fullNdfCachePath
}

// GetPartialNdfCachePath returns the partial NDF cache file path
func GetPartialNdfCachePath() string {
	return partialNdfCachePath
}

// saveNdfToCache saves NDF data to disk cache atomically using temp file + rename pattern
func saveNdfToCache(ndfData []byte, cachePath string) error {
	// Extract directory from cache path
	dir := filepath.Dir(cachePath)

	// Log directory creation attempt
	jww.DEBUG.Printf("CACHENDF-Checking cache directory: %s", dir)

	// Create directory if it doesn't exist
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		jww.WARN.Printf("CACHENDF-Failed to create cache directory: %+v", err)
		return errors.WithMessage(err, "failed to create cache directory")
	}

	// Create temporary file path
	tmpPath := cachePath + ".tmp"
	jww.DEBUG.Printf("CACHENDF-Writing to temp file: %s", tmpPath)

	// Write NDF data to temp file with permissions 0644
	err = os.WriteFile(tmpPath, ndfData, 0644)
	if err != nil {
		jww.WARN.Printf("CACHENDF-Failed to write temp file: %+v", err)
		return errors.WithMessage(err, "failed to write temp cache file")
	}

	jww.DEBUG.Printf("CACHENDF-Wrote %d bytes to temp file", len(ndfData))

	// Atomically rename temp file to final cache path
	err = os.Rename(tmpPath, cachePath)
	if err != nil {
		// Clean up temp file on failure
		os.Remove(tmpPath)
		jww.WARN.Printf("CACHENDF-Failed to rename temp file to cache path: %+v", err)
		return errors.WithMessage(err, "failed to atomically rename cache file")
	}

	jww.INFO.Printf("CACHENDF-Successfully cached NDF to %s (%d bytes)", cachePath, len(ndfData))
	return nil
}

// LoadNdfFromCache loads NDF data from disk cache
func LoadNdfFromCache(cachePath string) ([]byte, error) {
	jww.DEBUG.Printf("CACHENDF-Attempting to load NDF from cache: %s", cachePath)

	// Check if file exists
	fileInfo, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			jww.DEBUG.Printf("CACHENDF-Cache file does not exist: %s", cachePath)
		} else {
			jww.DEBUG.Printf("CACHENDF-Cache miss for %s: %+v", cachePath, err)
		}
		return nil, err
	}

	jww.DEBUG.Printf("CACHENDF-Cache file found: %s (%d bytes)", cachePath, fileInfo.Size())

	// Read entire file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		jww.DEBUG.Printf("CACHENDF-Failed to read cache file: %+v", err)
		return nil, errors.WithMessage(err, "failed to read cache file")
	}

	// Validate non-empty
	if len(data) == 0 {
		jww.DEBUG.Printf("CACHENDF-Cache file is empty: %s", cachePath)
		return nil, errors.New("cache file is empty")
	}

	jww.INFO.Printf("CACHENDF-Successfully loaded NDF from cache: %s (%d bytes)", cachePath, len(data))
	return data, nil
}

// RegisterNode performs the Node registration with the network
func RegisterNode(def *internal.Definition, instance *internal.Instance) error {
	// We don't check validity here, because the registration server should.
	node, nodePort, err := net.SplitHostPort(def.PublicAddress)
	if err != nil {
		return errors.Errorf("Failed to split host and port of public address "+
			"%s: %+v", def.PublicAddress, err)
	}
	nodePortInt, _ := strconv.ParseUint(nodePort, 10, 32)

	// Get the gateway's address
	gwAddr, _ := instance.GetGatewayData()
	// Split the address into port and address for message
	gwIP, gwPortStr, err := net.SplitHostPort(gwAddr)
	if err != nil {
		return errors.Errorf("Unable to parse gateway's address [%s]. Is it set up correctly?", gwAddr)
	}

	// Convert port to int to conform to message type
	gwPort, err := strconv.Atoi(gwPortStr)
	if err != nil {
		return errors.Errorf("Unable to parse gateway's port. Is it set up correctly?")
	}

	registrationRequest := &pb.NodeRegistration{
		Salt:             def.Salt,
		ServerTlsCert:    string(def.TlsCert),
		GatewayTlsCert:   string(def.Gateway.TlsCert),
		GatewayAddress:   gwIP, // FIXME (Jonah): this is inefficient, but will work for now
		GatewayPort:      uint32(gwPort),
		ServerAddress:    node,
		ServerPort:       uint32(nodePortInt),
		RegistrationCode: def.RegistrationCode,
	}

	// Construct sender interface
	sendFunc := func(h *connect.Host) (interface{}, error) {
		jww.DEBUG.Printf("Sending registration messages")
		return nil, instance.GetNetwork().
			SendNodeRegistration(h, registrationRequest)
	}

	sender := Sender{
		Send: sendFunc,
		Name: "RegisterNode",
	}

	// Attempt Node registration
	authHost, _ := instance.GetNetwork().GetHost(&id.Authorizer)
	_, err = Send(sender, instance, authHost)
	if err != nil {
		return errors.Errorf("Unable to send %s: %+v", sender.Name, err)
	}

	return nil
}

// Poll is used to retrieve updated state information from permissioning
//
//	and update our internal state accordingly
func Poll(instance *internal.Instance) error {

	// Fetch the host information from the network
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		return errors.New("Could not get permissioning host")
	}

	//get any skipped state reports
	reportedActivity := instance.GetStateMachine().GetActivityToReport()

	// Ping permissioning for updated information
	err := errors.New("dummy")
	var permResponse *pb.PermissionPollResponse
	for i := 0; i < 3 && err != nil; i++ {
		permResponse, err = PollPermissioning(permHost, instance, reportedActivity)
		if err != nil {
			if strings.Contains(err.Error(), "requires the Node not be assigned a round") ||
				strings.Contains(err.Error(), "requires the Node's be assigned a round") ||
				strings.Contains(err.Error(), "requires the Node be assigned a round") ||
				strings.Contains(err.Error(), "invalid transition") {
				instance.ReportNodeFailure(err)
			} else if strings.Contains(err.Error(), "Node cannot submit a rounderror when it is not") {
				err = nil
				break
			}
		}
		if err != nil {
			time.Sleep(1 * time.Second)
		}
	}

	if err != nil {
		return err
	}

	// Once done and in a completed state, manually switch back into waiting
	if reportedActivity == current.COMPLETED {
		// Sends the signal that this node is no longer in a round,
		// and thus the node is ready to be killed. Signal is sent only if a SIGINT
		// has already been sent.
		select {
		case killed := <-instance.GetKillChan():
			killed <- struct{}{}
		default:
		}
		ok, err := instance.GetStateMachine().Update(current.WAITING)
		if err != nil || !ok {
			return errors.Errorf("Could not transition to WAITING state: %v", err)
		}
	}

	//updates the NDF with changes
	if permResponse != nil {
		err = UpdateNDf(permResponse, instance)
		if err != nil {
			return errors.WithMessage(err, "Failed to update the NDFs")
		}

		// Update the internal state of rounds and the state machine
		err = UpdateRounds(permResponse, instance)

		// Update the earliest round store for gateway passage
		if permResponse.EarliestRoundErr == "" {
			instance.SetEarliestRound(permResponse.EarliestClientRound, permResponse.EarliestGatewayRound,
				permResponse.EarliestRoundTimestamp)
		}
	} else {
		jww.WARN.Printf("Skipped processing poll due to nul permResponse")
	}

	return err
}

// PollPermissioning  the permissioning server for updates
func PollPermissioning(permHost *connect.Host, instance *internal.Instance,
	reportedActivity current.Activity) (*pb.PermissionPollResponse, error) {
	var fullNdfHash, partialNdfHash []byte

	// Get the ndf hashes for the full ndf if available
	if instance.GetNetworkStatus().GetFullNdf() != nil {
		fullNdfHash = instance.GetNetworkStatus().GetFullNdf().GetHash()
	}

	// Get the ndf hashes for the partial ndf if available
	if instance.GetNetworkStatus().GetPartialNdf() != nil {
		partialNdfHash = instance.GetNetworkStatus().GetPartialNdf().GetHash()
	}

	// Get the update id and activity of the state machine
	lastUpdateId := instance.GetNetworkStatus().GetLastUpdateID()

	// The ring buffer returns negative none but message type doesn't support signed numbers
	// fixme: maybe make proto have signed ints
	if lastUpdateId == -1 {
		lastUpdateId = 0
	}

	var clientReport []*pb.ClientError
	var err error
	latestRound := instance.GetRoundManager().GetCurrentRound()
	if reportedActivity == current.COMPLETED {
		clientReport, err = instance.GetClientReport().Receive(latestRound)
		if err != nil {
			jww.ERROR.Printf("Unable to receive client report: %+v", err)
		}
		if len(clientReport) > 0 {
			jww.WARN.Printf("Client error reports found for"+
				" round %v: %d reports found", latestRound, len(clientReport))
		}

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

		ServerAddress: instance.GetIP(),
		ServerVersion: instance.GetServerVersion(),
		ClientErrors:  clientReport,
	}

	ed := instance.GetEd25519Key()
	if ed != nil {
		pollMsg.Ed25519 = ed.Bytes()
	}

	if reportedActivity == current.ERROR {
		pollMsg.Error = instance.GetRecoveredError()
		jww.INFO.Printf("Reporting error to permissioning: %+v", pollMsg.Error)
		instance.ClearRecoveredError()
		if instance.GetStateMachine().Get() == current.ERROR {
			ok, err := instance.GetStateMachine().Update(current.WAITING)
			if err != nil || !ok {
				err = errors.WithMessage(err, "Could not move to waiting state to recover from error")
				return nil, err
			}
		}
	}

	// Construct sender interface
	sendFunc := func(h *connect.Host) (interface{}, error) {
		return instance.GetNetwork().SendPoll(permHost, pollMsg)
	}

	sender := Sender{
		Send: sendFunc,
		Name: "Poll",
	}

	// Attempt to send Permissioning Poll
	authHost, _ := instance.GetNetwork().GetHost(&id.Authorizer)
	face, err := Send(sender, instance, authHost)
	if err != nil {
		return nil, errors.Errorf("Unable to send %s: %+v", sender.Name, err)
	}

	// Process response
	permissioningResponse := face.(*pb.PermissionPollResponse)
	return permissioningResponse, err
}

// queueUntilRealtime is an internal function that transitions the instance
// state from QUEUED/STANDBY to REALTIME at the provided start time.
// If the start time is BEFORE the current time, it starts immediately and
// prints a warning regarding possible clock skew.
func queueUntilRealtime(instance *internal.Instance, start time.Time) {
	// Check if the start time has already past
	now := time.Now()
	if now.After(start) {
		jww.WARN.Printf("Possible clock skew detected when queuing "+
			"for realtime , %s is after %s", now, start)
		now = start
	}

	// Sleep until start time
	until := start.Sub(now)
	jww.INFO.Printf("Sleeping for %dms for realtime start",
		until.Milliseconds())
	time.Sleep(until)

	// Update to realtime when ready
	ok, err := instance.GetStateMachine().Update(current.REALTIME)
	if !ok || err != nil {
		jww.FATAL.Panicf("Cannot move to realtime state: %+v", err)
	}
}

// UpdateRounds processes the polling response from permissioning for round updates,
// installing any round changes if needed. It also parsed the message and
// determines where to transition given context
func UpdateRounds(permissioningResponse *pb.PermissionPollResponse, instance *internal.Instance) error {

	// Parse the response for updates
	newUpdates := permissioningResponse.Updates

	//skip all processing of round updates if the node knows of no round updates
	//which is normally the result of a crash and restart
	skipUpdates := instance.IsFirstPoll() && !instance.GetFirstRun()
	// Parse the round info updates if they exist
	for _, roundInfo := range newUpdates {
		// Add the new information to the network instance
		_, err := instance.GetNetworkStatus().RoundUpdate(roundInfo)
		if err != nil {
			if strings.Contains(err.Error(), "id is older than first tracked") {
				continue
			}
			return errors.Errorf("Unable to update for round %+v: %+v", roundInfo.ID, err)
		}

		//skip all round updates older than those known about. this can happen as
		if skipUpdates {
			continue
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
				// Do nothing
			case states.PRECOMPUTING: // Prepare for precomputing state

				// Standby until in WAITING state to ensure a valid transition into precomputing
				curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.WAITING)
				if curActivity != current.WAITING || err != nil {
					return errors.Errorf("Cannot start precomputing when not in waiting state for round %v: %+v",
						roundInfo.ID, err)
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
				duration := time.Unix(0, int64(roundInfo.Timestamps[states.QUEUED]))
				go queueUntilRealtime(instance, duration)

			case states.REALTIME:
				// Don't do anything

			case states.COMPLETED:

			case states.FAILED:
				errStr := "Unknown error"
				firstSource := &id.Permissioning
				if roundInfo.Errors != nil {
					var err error
					firstSource, err = id.Unmarshal(roundInfo.Errors[0].NodeId)
					var idStr string
					if err != nil {
						idStr = "BAD ID"
					} else {
						idStr = firstSource.String()
					}

					errStr = fmt.Sprintf("%s first failed %v: %s", idStr, roundInfo.ID, roundInfo.Errors[0].Error)
				}

				r, err := instance.GetRoundManager().GetRound(id.Round(roundInfo.ID))

				rid := id.Round(roundInfo.ID)

				// if the round is unknown, restart with an error unless the report
				// is from a node that did not know about the round. Do not restart if
				// the node didnt know because that can cause ping ponging restarts
				if err != nil {
					jww.WARN.Printf("Received secondary participatory fail in round %v which node has no knowledge of", roundInfo.ID)
					continue
				}

				// do nothing if you have the round as complete
				if r.GetCurrentPhaseType() == phase.Complete && r.GetID() == rid {
					jww.WARN.Printf("Received participatory fail in round %v which node has completed", rid)
					return nil
					// fail if the round is in progress
				} else if r.GetCurrentPhaseType() != phase.Complete && r.GetID() == rid {
					jww.WARN.Printf("Received participatory fail in round %v which is in progress", roundInfo.ID)
					instance.ReportRoundFailure(errors.Errorf("Notified of round failure for participatory round: %s", errStr),
						instance.GetID(), rid)
					// if the node is working on a different round, do nothing, the error is likely old
				} else {
					jww.WARN.Printf("Received participatory fail from old round %v", rid)
				}

			default:
				return errors.Errorf("Round in unknown state: %v", states.Round(roundInfo.State))

			}

		}
	}
	return nil
}

// hostsRegistered tracks whether hosts have been registered from the NDF (0 = not yet, 1 = done)
var hostsRegistered uint32

// UpdateNDf processes the polling response from permissioning for ndf updates,
// installing any ndf changes if needed and connecting to new nodes. Also saves
// a list of node addresses found in the NDF to a separate file.
func UpdateNDf(permissioningResponse *pb.PermissionPollResponse, instance *internal.Instance) error {
	if permissioningResponse.FullNDF != nil {
		// Log that cached hash doesn't match - new NDF is being downloaded
		if instance.GetNetworkStatus().GetFullNdf() != nil {
			oldHash := base64.StdEncoding.EncodeToString(instance.GetNetworkStatus().GetFullNdf().GetHash())
			jww.INFO.Printf("CACHENDF-Cached full NDF hash does not match current NDF - Downloading (cached: %s)", oldHash)
		}

		// Update the full ndf
		err := instance.GetNetworkStatus().UpdateFullNdf(permissioningResponse.FullNDF)
		if err != nil {
			return errors.Errorf("Could not update full ndf: %+v", err)
		}

		// Cache the full NDF
		jww.DEBUG.Printf("CACHENDF-Caching full NDF after successful update")
		err = saveNdfToCache(permissioningResponse.FullNDF.Ndf, fullNdfCachePath)
		if err != nil {
			jww.WARN.Printf("CACHENDF-Failed to cache full NDF: %+v", err)
			// Continue execution - cache failure is non-fatal
		}

		// Save the list of node IP addresses to file
		err = SaveNodeIpList(instance.GetNetworkStatus().GetFullNdf().Get(),
			instance.GetDefinition().IpListOutput, instance.GetDefinition().ID)
		if err != nil {
			jww.ERROR.Printf("Failed to save list of IP addresses from NDF: %v", err)
		}

		jww.INFO.Printf("New NDF Received, hash: %s", base64.StdEncoding.EncodeToString(instance.GetNetworkStatus().GetFullNdf().GetHash()))
	} else {
		// No new NDF sent - cached hash matched current NDF
		if instance.GetNetworkStatus().GetFullNdf() != nil {
			currentHash := base64.StdEncoding.EncodeToString(instance.GetNetworkStatus().GetFullNdf().GetHash())
			jww.DEBUG.Printf("CACHENDF-Cached full NDF hash matches current NDF - Skipping download (hash: %s)", currentHash)
		}
	}

	if permissioningResponse.PartialNDF != nil {
		// Log that cached hash doesn't match - new partial NDF is being downloaded
		if instance.GetNetworkStatus().GetPartialNdf() != nil {
			oldHash := base64.StdEncoding.EncodeToString(instance.GetNetworkStatus().GetPartialNdf().GetHash())
			jww.INFO.Printf("CACHENDF-Cached partial NDF hash does not match current NDF - Downloading (cached: %s)", oldHash)
		}

		// Update the partial ndf
		err := instance.GetNetworkStatus().UpdatePartialNdf(permissioningResponse.PartialNDF)
		if err != nil {
			return errors.Errorf("Could not update partial ndf: %+v", err)
		}

		// Cache the partial NDF
		jww.DEBUG.Printf("CACHENDF-Caching partial NDF after successful update")
		err = saveNdfToCache(permissioningResponse.PartialNDF.Ndf, partialNdfCachePath)
		if err != nil {
			jww.WARN.Printf("CACHENDF-Failed to cache partial NDF: %+v", err)
			// Continue execution - cache failure is non-fatal
		}
	} else {
		// No new partial NDF sent - cached hash matched current NDF
		if instance.GetNetworkStatus().GetPartialNdf() != nil {
			currentHash := base64.StdEncoding.EncodeToString(instance.GetNetworkStatus().GetPartialNdf().GetHash())
			jww.DEBUG.Printf("CACHENDF-Cached partial NDF hash matches current NDF - Skipping download (hash: %s)", currentHash)
		}
	}

	// If we have a new NDF or if updates are present (implying successful connectivity check),
	// we should update connections and register hosts if needed.
	hasNdf := permissioningResponse.PartialNDF != nil || permissioningResponse.FullNDF != nil
	hasUpdates := len(permissioningResponse.Updates) > 0

	if hasNdf || hasUpdates {

		// Update the nodes in the network.Instance with the new ndf
		err := instance.GetNetworkStatus().UpdateNodeConnections()
		if err != nil {
			return errors.Errorf("Could not update node connections: %+v", err)
		}

		// LAZY HOST REGISTRATION FIX:
		// After the first successful poll that returns an NDF (connectivity verified),
		// register hosts from the current NDF. This is done using atomic compare-and-swap
		// to ensure it only happens once.
		// By deferring host registration until after connectivity passes (when permissioning
		// actually returns an NDF), we prevent network saturation during the vetting process
		// that can cause checkConnectivity to timeout.
		if atomic.CompareAndSwapUint32(&hostsRegistered, 0, 1) {
			fullNdf := instance.GetNetworkStatus().GetFullNdf()
			if fullNdf != nil && fullNdf.Get() != nil {
				ndfData := fullNdf.Get()
				jww.INFO.Printf("LAZY-HOST-REG: First NDF received (connectivity verified) - registering hosts from current NDF (%d nodes)", len(ndfData.Nodes))

				registered := 0
				for _, node := range ndfData.Nodes {
					nodeId, err := id.Unmarshal(node.ID)
					if err != nil {
						jww.DEBUG.Printf("LAZY-HOST-REG: Could not unmarshal node ID: %v", err)
						continue
					}

					// Skip our own node - it's already added separately
					if nodeId.Cmp(instance.GetDefinition().ID) {
						continue
					}

					// Add host to the network's host manager
					_, err = instance.GetNetwork().AddHost(nodeId, node.Address, []byte(node.TlsCertificate),
						connect.GetDefaultHostParams())
					if err != nil {
						// Log but continue - some hosts might be unreachable or already added
						jww.DEBUG.Printf("LAZY-HOST-REG: Could not add host %s at %s: %v",
							nodeId, node.Address, err)
						continue
					}
					registered++
				}

				jww.INFO.Printf("LAZY-HOST-REG: Successfully registered %d/%d hosts from NDF after connectivity verification",
					registered, len(ndfData.Nodes))
			} else {
				jww.WARN.Printf("LAZY-HOST-REG: First NDF received but full NDF is not available for host registration")
			}
		}
	}

	return nil

}

// FindSelfInNdf parses the ndf to determine if we exist in the ndf.
func FindSelfInNdf(def *internal.Definition, newNdf *ndf.NetworkDefinition) bool {
	// Find this node's place in the newNDF
	for _, newNode := range newNdf.Nodes {
		// If we exist in the ndf, return no error
		if bytes.Equal(newNode.ID, def.ID.Bytes()) {
			return true
		}
	}
	return false
}
