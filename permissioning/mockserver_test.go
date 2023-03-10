////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	crand "crypto/rand"
	"fmt"
	"gitlab.com/elixxir/comms/authorizer"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
)

var count = 0
var countLock sync.Mutex

// --------------------------------Dummy implementation of authorizer------------------------------------------
type mockAuthorizer struct{}

func (a *mockAuthorizer) Authorize(auth *pb.AuthorizerAuth, ipAddr string) (err error) {
	return nil
}

func (a *mockAuthorizer) RequestCert(msg *pb.AuthorizerCertRequest) (*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func (a *mockAuthorizer) RequestEABCredentials(msg *pb.EABCredentialRequest) (*pb.EABCredentialResponse, error) {
	return &pb.EABCredentialResponse{}, nil
}

type mockAuthorizerErrorPath struct{}

func (a *mockAuthorizerErrorPath) Authorize(auth *pb.AuthorizerAuth, ipAddr string) (err error) {
	return errors.Errorf("Could not authorize")
}

func (a *mockAuthorizerErrorPath) RequestCert(msg *pb.AuthorizerCertRequest) (*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func (a *mockAuthorizerErrorPath) RequestEABCredentials(msg *pb.EABCredentialRequest) (*pb.EABCredentialResponse, error) {
	return &pb.EABCredentialResponse{}, nil
}

func startAuthorizer(addr, nAddr string, nodeId *id.ID, cert, key []byte) (*authorizer.Comms, error) {
	mockHandler := authorizer.Handler(&mockAuthorizer{})
	authComms := authorizer.StartAuthorizerServer(&id.Authorizer, addr, mockHandler, cert, key)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	_, err := authComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Authorizer could not connect to node: %v", err)
	}

	return authComms, nil
}

func startAuthorizerErrorPath(addr, nAddr string, nodeId *id.ID, cert, key []byte) (*authorizer.Comms, error) {
	mockHandler := authorizer.Handler(&mockAuthorizerErrorPath{})
	authComms := authorizer.StartAuthorizerServer(&id.Authorizer, addr, mockHandler, cert, key)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	_, err := authComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Authorizer could not connect to node: %v", err)
	}

	return authComms, nil
}

// --------------------------------Dummy implementation of permissioning server --------------------------------
type mockPermission struct {
	cert []byte
	key  []byte
}

func (i *mockPermission) PollNdf([]byte) (*pb.NDF, error) {
	return nil, nil
}

func (i *mockPermission) RegisterUser(clientRegistration *mixmessages.ClientRegistration) (*mixmessages.SignedClientRegistrationConfirmations, error) {
	return nil, nil
}

func (i *mockPermission) RegisterNode([]byte, string, string, string, string, string) error {
	return nil
}

func (i *mockPermission) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
	ourNdf := testUtil.NDF
	fullNdf, _ := ourNdf.Marshal()
	stripNdf, _ := ourNdf.StripNdf().Marshal()

	fullNDFMsg := &pb.NDF{Ndf: fullNdf}
	partialNDFMsg := &pb.NDF{Ndf: stripNdf}

	err := signNdf(fullNDFMsg, i.key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign full ndf: %+v", err)
	}
	err = signNdf(partialNDFMsg, i.key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign partial ndf: %+v", err)
	}

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

// --------------------------------Dummy implementation of permissioning server w/ connection error  ----------------

type mockPermission_ConnectionError struct {
	cert []byte
	key  []byte
}

func (i *mockPermission_ConnectionError) PollNdf([]byte) (*pb.NDF, error) {
	return nil, errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) RegisterUser(clientRegistration *mixmessages.ClientRegistration) (*mixmessages.SignedClientRegistrationConfirmations, error) {
	return nil, errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) RegisterNode([]byte, string, string, string, string, string) error {
	return errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {

	return nil, errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	return nil, errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) GetCurrentClientVersion() (string, error) {
	return "0.0.0", errors.Errorf("connection refused")
}

func (i *mockPermission_ConnectionError) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	return nil, errors.Errorf("connection refused")
}

// --------------------------------Dummy implementation of permissioning server w/ connection error  ----------------

type mockPermission_ConnectionErrorOnce struct {
	count int
	cert  []byte
	key   []byte
}

func (i *mockPermission_ConnectionErrorOnce) PollNdf([]byte) (*pb.NDF, error) {
	if i.count == 0 {
		i.count++
		return nil, errors.Errorf("connection refused")
	}
	return nil, nil
}

func (i *mockPermission_ConnectionErrorOnce) RegisterUser(clientRegistration *mixmessages.ClientRegistration) (*mixmessages.SignedClientRegistrationConfirmations, error) {
	if i.count == 0 {
		i.count++
		return nil, errors.Errorf("connection refused")
	}
	return nil, nil
}

func (i *mockPermission_ConnectionErrorOnce) RegisterNode([]byte, string, string, string, string, string) error {
	if i.count == 0 {
		i.count++
		return errors.Errorf("connection refused")
	}
	return nil
}

func (i *mockPermission_ConnectionErrorOnce) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
	if i.count == 0 {
		i.count++
		return nil, errors.Errorf("connection refused")
	}
	return nil, nil
}

func (i *mockPermission_ConnectionErrorOnce) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	if i.count == 0 {
		i.count++
		return nil, errors.Errorf("connection refused")
	}
	return nil, nil
}

func (i *mockPermission_ConnectionErrorOnce) GetCurrentClientVersion() (string, error) {
	if i.count == 0 {
		i.count++
		return "nil", errors.Errorf("connection refused")
	}
	return "", nil
}

func (i *mockPermission_ConnectionErrorOnce) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	if i.count == 0 {
		i.count++
		return nil, errors.Errorf("connection refused")
	}
	return nil, nil
}

// --------------------------------Dummy implementation of permissioning server --------------------------------
type mockPermissionMultipleRounds struct {
	cert []byte
	key  []byte
}

func (i *mockPermissionMultipleRounds) PollNdf([]byte) (*pb.NDF, error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) RegisterUser(clientRegistration *mixmessages.ClientRegistration) (*mixmessages.SignedClientRegistrationConfirmations, error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) RegisterNode([]byte, string, string, string, string, string) error {
	return nil
}

func (i *mockPermissionMultipleRounds) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	return nil, nil
}

func (i *mockPermissionMultipleRounds) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
	ourNdf := testUtil.NDF
	fullNdf, _ := ourNdf.Marshal()
	stripNdf, _ := ourNdf.StripNdf().Marshal()

	fullNDFMsg := &pb.NDF{Ndf: fullNdf}
	partialNDFMsg := &pb.NDF{Ndf: stripNdf}

	err := signNdf(fullNDFMsg, i.key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign full ndf: %+v", err)
	}
	err = signNdf(partialNDFMsg, i.key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign partial ndf: %+v", err)
	}

	ourRoundInfoList, err := buildRoundInfoMessages(i.key)
	if err != nil {
		return nil, errors.Errorf("Failed to build round info message: %+v", err)
	}

	return &pb.PermissionPollResponse{
		FullNDF:    fullNDFMsg,
		PartialNDF: partialNDFMsg,
		Updates:    ourRoundInfoList,
	}, nil
}

func buildRoundInfoMessages(key []byte) ([]*pb.RoundInfo, error) {
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
	err := signRoundInfo(precompRoundInfo, key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign precomp round info: %+v", err)
	}

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
	err = signRoundInfo(standbyRoundInfo, key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign standby round info: %+v", err)
	}

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
	err = signRoundInfo(newNodeRoundInfo, key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign new node round info: %+v", err)
	}

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
	err = signRoundInfo(realtimeRoundInfo, key)
	if err != nil {
		return nil, errors.Errorf("Failed to sign realtime round info: %+v", err)
	}

	return []*pb.RoundInfo{precompRoundInfo, standbyRoundInfo, newNodeRoundInfo, realtimeRoundInfo}, nil
}

func (i *mockPermissionMultipleRounds) GetCurrentClientVersion() (string, error) {
	return "0.0.0", nil
}

func (i *mockPermissionMultipleRounds) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
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

// Utility function which signs an ndf message
func signNdf(ourNdf *pb.NDF, key []byte) error {
	pk, err := tls.LoadRSAPrivateKey(string(key))
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	err = signature.SignRsa(ourNdf, ourPrivKey)
	if err != nil {
		return errors.Errorf("Failed to sign ndf: %+v", err)
	}

	return nil
}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo, key []byte) error {
	pk, err := tls.LoadRSAPrivateKey(string(key))
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	err = signature.SignRsa(ri, ourPrivKey)
	if err != nil {
		return errors.Errorf("Failed to sign round info: %+v", err)
	}
	return nil
}

// Utility function which builds a signed full-ndf message
func setupFullNdf(key []byte) (*pb.NDF, error) {
	pk, err := tls.LoadRSAPrivateKey(string(key))
	if err != nil {
		return nil, errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}
	tmpNdf, err := ndf.Unmarshal(testUtil.ExampleNDF)
	if err != nil {
		return nil, errors.Errorf("Failed to decode NDF: %+v", err)
	}
	f.Ndf, err = tmpNdf.Marshal()
	if err != nil {
		return nil, errors.Errorf("Failed to marshal ndf: %+v", err)
	}

	err = signature.SignRsa(f, ourPrivKey)

	return f, nil
}

// Utility function which builds a signed partial-ndf message
func setupPartialNdf(key []byte) (*pb.NDF, error) {
	pk, err := tls.LoadRSAPrivateKey(string(key))
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

	err = signature.SignRsa(f, ourPrivKey)

	return f, nil
}

// Utility function which creates an instance
func createServerInstance(t *testing.T) (instance *internal.Instance, pAddr,
	nodeAddr string, nodeId *id.ID, cert, key []byte, err error) {
	cert, _ = utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ = utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewIdFromUInt(uint64(0), id.Node, t)
	countLock.Lock()
	nodeAddr = fmt.Sprintf("0.0.0.0:%d", 7200+count)
	pAddr = fmt.Sprintf("0.0.0.0:%d", 2200+count)
	count++
	countLock.Unlock()
	// Build the node
	emptyNdf := builEmptydMockNdf()
	// Initialize definition
	def := &internal.Definition{
		RngStreamGen: fastRNG.NewStreamGenerator(8, 8, csprng.NewSystemRNG),

		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        nil,
		PrivateKey:       nil,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		LogPath:          "",
		MetricLogPath:    "",
		Network: internal.Perm{
			TlsCert: cert,
			Address: pAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
		DevMode:         true,
	}
	def.Gateway.ID = nodeId.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	def.PrivateKey, _ = rsa.GenerateKey(crand.Reader, 1024)

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		return
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err = internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		return
	}

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		return
	}

	err = nil
	return
}

// Utility function which starts up a permissioning server
func startPermissioning(pAddr, nAddr string, nodeId *id.ID, cert, key []byte) (*registration.Comms, error) {
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermission{
		cert: cert,
		key:  key,
	})
	permComms := registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key, nil)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err := permComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil
}

func startMultipleRoundUpdatesPermissioning(pAddr, nAddr string, nodeId *id.ID, cert, key []byte) (*registration.Comms, error) {
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermissionMultipleRounds{
		cert: cert,
		key:  key,
	})
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	permComms := registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key, nil)
	_, err := permComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil

}

func startPermissioning_ConnectionError(pAddr, nAddr string, nodeId *id.ID, cert, key []byte) (*registration.Comms, error) {
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermission_ConnectionError{
		cert: cert,
		key:  key,
	})
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	permComms := registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key, nil)
	_, err := permComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil

}

func startPermissioning_ConnectionErrorOnce(pAddr, nAddr string, nodeId *id.ID, cert, key []byte) (*registration.Comms, error) {
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermission_ConnectionErrorOnce{
		cert: cert,
		key:  key,
	})
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	permComms := registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key, nil)
	_, err := permComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, nil
}
