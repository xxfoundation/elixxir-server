///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	crand "crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/gateway"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"math/rand"
	"testing"
	"time"
)

var nodeId *id.ID
var permComms *registration.Comms
var gwComms *gateway.Comms
var testNdf *ndf.NetworkDefinition
var pAddr string
var cnt = 0
var nodeAddr string

// --------------------------------Dummy implementation of permissioning server --------------------------------
type mockPermission struct{}

func (i *mockPermission) PollNdf([]byte, *connect.Auth) ([]byte, error) {
	return nil, nil
}

func (i *mockPermission) RegisterUser(registrationCode, test string) (hash []byte, err error) {
	return nil, nil
}

func (i *mockPermission) RegisterNode(*id.ID, string, string, string, string, string) error {
	return nil
}

func (i *mockPermission) Poll(*pb.PermissioningPoll, *connect.Auth, string) (*pb.PermissionPollResponse, error) {
	ourNdf := testUtil.NDF
	fullNdf, _ := ourNdf.Marshal()
	stripNdf, _ := ourNdf.StripNdf().Marshal()

	fullNDFMsg := &pb.NDF{Ndf: fullNdf}
	partialNDFMsg := &pb.NDF{Ndf: stripNdf}

	signNdf(fullNDFMsg)
	signNdf(partialNDFMsg)

	return &pb.PermissionPollResponse{
		FullNDF:    fullNDFMsg,
		PartialNDF: partialNDFMsg,
	}, nil
}

func (i *mockPermission) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	return nil, nil
}

func (i *mockPermission) GetCurrentClientVersion() (string, error) {
	return "0.0.0", nil
}

func (i *mockPermission) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	return nil, nil
}

// --------------------------------Dummy implementation of permissioning server --------------------------------
type mockPermissionMultipleRounds struct{}

func (i *mockPermissionMultipleRounds) PollNdf([]byte, *connect.Auth) ([]byte, error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) RegisterUser(registrationCode, test string) (hash []byte, err error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) RegisterNode(*id.ID, string, string, string, string, string) error {
	return nil
}

func (i *mockPermissionMultipleRounds) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) Poll(*pb.PermissioningPoll, *connect.Auth, string) (*pb.PermissionPollResponse, error) {
	ourNdf := testUtil.NDF
	fullNdf, _ := ourNdf.Marshal()
	stripNdf, _ := ourNdf.StripNdf().Marshal()

	fullNDFMsg := &pb.NDF{Ndf: fullNdf}
	partialNDFMsg := &pb.NDF{Ndf: stripNdf}

	signNdf(fullNDFMsg)
	signNdf(partialNDFMsg)

	ourRoundInfoList := buildRoundInfoMessages()

	return &pb.PermissionPollResponse{
		FullNDF:    fullNDFMsg,
		PartialNDF: partialNDFMsg,
		Updates:    ourRoundInfoList,
	}, nil
}

func buildRoundInfoMessages() []*pb.RoundInfo {
	numUpdates := uint64(0)

	node1 := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	node2 := []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	node3 := []byte{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	node4 := []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	now := time.Now()
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PRECOMPUTING] = uint64(now.UnixNano())
	timestamps[states.REALTIME] = uint64(time.Now().Add(500 * time.Millisecond).UnixNano())

	// Create a topology for round info
	jww.FATAL.Println(node1)
	ourTopology := [][]byte{node1, node2, node3}

	// Construct round info message indicating PRECOMP starting
	precompRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   numUpdates,
		State:      uint32(states.PRECOMPUTING),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Mocking permissioning server signing message
	signRoundInfo(precompRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Construct round info message indicating STANDBY starting
	standbyRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   numUpdates,
		State:      uint32(states.STANDBY),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Mocking permissioning server signing message
	signRoundInfo(standbyRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Construct message which adds node to team
	ourTopology = append(ourTopology, node4)

	// Add new round in standby stage
	newNodeRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   numUpdates,
		State:      uint32(states.STANDBY),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Set the signature field of the round info
	signRoundInfo(newNodeRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Construct round info message for REALTIME
	realtimeRoundInfo := &pb.RoundInfo{
		ID:         0,
		UpdateID:   numUpdates,
		State:      uint32(states.REALTIME),
		Topology:   ourTopology,
		Timestamps: timestamps,
	}

	// Increment updates id for next message
	numUpdates++

	// Set the signature field of the round info
	signRoundInfo(realtimeRoundInfo)

	return []*pb.RoundInfo{precompRoundInfo, standbyRoundInfo, newNodeRoundInfo, realtimeRoundInfo}
}

func (i *mockPermissionMultipleRounds) GetCurrentClientVersion() (string, error) {
	return "0.0.0", nil
}

func (i *mockPermissionMultipleRounds) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	return nil, nil
}

// --------------------------Dummy implementation of gateway server --------------------------------------
type mockGateway struct{}

func (*mockGateway) CheckMessages(userID *id.ID, messageID string, ipAddress string) ([]string, error) {
	return nil, nil
}

func (*mockGateway) GetMessage(userID *id.ID, msgID string, ipAddress string) (*pb.Slot, error) {
	return nil, nil
}

func (*mockGateway) PutMessage(message *pb.Slot, ipAddress string) error {
	return nil
}

func (*mockGateway) RequestNonce(message *pb.NonceRequest, ipAddress string) (*pb.Nonce, error) {
	return nil, nil
}

func (*mockGateway) ConfirmNonce(message *pb.RequestRegistrationConfirmation, ipAddress string) (*pb.
	RegistrationConfirmation, error) {
	return nil, nil
}

func (*mockGateway) PollForNotifications(auth *connect.Auth) ([]*id.ID, error) {
	return nil, nil
}

func (*mockGateway) Poll(*pb.GatewayPoll) (*pb.GatewayPollResponse, error) {
	return nil, nil
}

var dummyStates = [current.NUM_STATES]state.Change{
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
	func(from current.Activity) error { return nil },
}

func mockServerDef(i interface{}) *internal.Definition {
	nid := internal.GenerateId(i)

	resourceMetric := measure.ResourceMetric{
		Time:          time.Now(),
		MemAllocBytes: 0,
		NumThreads:    0,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(resourceMetric)

	def := internal.Definition{
		ID:              nid,
		ResourceMonitor: &resourceMonitor,
		FullNDF:         testUtil.NDF,
	}

	return &def
}

// ------------------------------ Utility functions for testing purposes  ----------------------------------------------

func builEmptydMockNdf() *ndf.NetworkDefinition {

	ourMockNdf := &ndf.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{},
		Gateways:  []ndf.Gateway{},
		UDB:       ndf.UDB{},
	}

	return ourMockNdf
}

func buildMockNdf(nodeId *id.ID, nodeAddress, gwAddress string, cert, key []byte) {
	node := ndf.Node{
		ID:             nodeId.Bytes(),
		TlsCertificate: string(cert),
		Address:        nodeAddress,
	}
	gw := ndf.Gateway{
		Address:        gwAddress,
		TlsCertificate: string(cert),
	}
	mockGroup := ndf.Group{
		Prime:      "25",
		SmallPrime: "42",
		Generator:  "2",
	}
	testNdf = &ndf.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{node},
		Gateways:  []ndf.Gateway{gw},
		E2E:       mockGroup,
		CMIX:      mockGroup,
		UDB:       ndf.UDB{},
	}
}

// Utility function which signs an ndf message
func signNdf(ourNdf *pb.NDF) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ourNdf, ourPrivKey)

	return nil
}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)
	return nil
}

// Utility function which builds a signed full-ndf message
func setupFullNdf() (*pb.NDF, error) {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return nil, errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}
	tmpNdf, _, _ := ndf.DecodeNDF(testUtil.ExampleJSON)
	f.Ndf, err = tmpNdf.Marshal()
	if err != nil {
		return nil, errors.Errorf("Failed to marshal ndf: %+v", err)
	}

	if err != nil {
		return nil, errors.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f, nil
}

// Utility function which builds a signed partial-ndf message
func setupPartialNdf() (*pb.NDF, error) {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return nil, errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}

	stipped, err := testUtil.NDF.StripNdf().Marshal()
	f.Ndf = stipped

	if err != nil {
		return nil, errors.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f, nil
}

// Utility function which creates an instance
func createServerInstance(t *testing.T) (*internal.Instance, error) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewIdFromUInt(uint64(0), id.Node, t)
	nodeAddr = fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000)+cnt)
	pAddr = fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
	cnt++
	// Build the node
	emptyNdf := builEmptydMockNdf()
	// Initialize definition
	def := &internal.Definition{
		Flags:         internal.Flags{},
		ID:            nodeId,
		PublicKey:     nil,
		PrivateKey:    nil,
		TlsCert:       cert,
		TlsKey:        key,
		Address:       nodeAddr,
		LogPath:       "",
		MetricLogPath: "",
		UserRegistry:  nil,
		Permissioning: internal.Perm{
			TlsCert: []byte(testUtil.RegCert),
			Address: pAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
	}
	def.Gateway.ID = nodeId.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	def.PrivateKey, _ = rsa.GenerateKey(crand.Reader, 1024)

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		return nil, errors.Errorf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm,
		"1.1.0")
	if err != nil {
		return nil, errors.Errorf("Unable to create instance: %+v", err)
	}

	// Add permissioning as a host
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Permissioning.Address,
		def.Permissioning.TlsCert, false, false)
	if err != nil {
		return nil, errors.Errorf("Failed to add permissioning host: %+v", err)
	}

	return instance, nil
}

// Utility function which starts up a permissioning server
func startPermissioning() (*registration.Comms, error) {

	cert := []byte(testUtil.RegCert)
	key := []byte(testUtil.RegPrivKey)
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId, pAddr, cert, false, false)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil
}

func startGateway() (*gateway.Comms, error) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	gAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	gHandler := gateway.Handler(&mockGateway{})
	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)
	gwComms = gateway.StartGateway(gwID, gAddr, gHandler, cert, key)
	_, err := gwComms.AddHost(nodeId, nodeAddr, cert, false, false)
	if err != nil {
		return nil, err
	}

	return gwComms, nil
}

func startMultipleRoundUpdatesPermissioning() (*registration.Comms, error) {
	cert := []byte(testUtil.RegCert)
	key := []byte(testUtil.RegPrivKey)
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermissionMultipleRounds{})
	permComms = registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId, pAddr, cert, false, false)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil

}
