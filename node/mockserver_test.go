///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	crand "crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
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
	"sync"
	"testing"
	"time"
)

var count = 0
var countLock sync.Mutex

// --------------------------------Dummy implementation of permissioning server --------------------------------
type mockPermission struct {
	err error
}

func (i *mockPermission) PollNdf([]byte) ([]byte, error) {
	return nil, i.err
}

func (i *mockPermission) RegisterUser(registrationCode, test, test2 string) (hash []byte, hash2 []byte, err error) {
	return nil, nil, i.err
}

func (i *mockPermission) RegisterNode([]byte, string, string, string, string, string) error {
	return i.err
}

func (i *mockPermission) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
	ourNdf := testUtil.NDF
	fullNdf, _ := ourNdf.Marshal()
	stripNdf, _ := ourNdf.StripNdf().Marshal()

	fullNDFMsg := &pb.NDF{Ndf: fullNdf}
	partialNDFMsg := &pb.NDF{Ndf: stripNdf}

	err := signNdf(fullNDFMsg)
	if err != nil {
		return nil, errors.Errorf("Failed to sign full Ndf: %+v", err)
	}
	err = signNdf(partialNDFMsg)
	if err != nil {
		return nil, errors.Errorf("Failed to sign partial NDF: %+v", err)
	}

	return &pb.PermissionPollResponse{
		FullNDF:    fullNDFMsg,
		PartialNDF: partialNDFMsg,
	}, i.err
}

func (i *mockPermission) CheckRegistration(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
	return &pb.RegisteredNodeConfirmation{
		IsRegistered: true,
	}, i.err
}

// Set an error that RPC calls to this permissioning server will return
func (i *mockPermission) SetDesiredError(err error) {
	i.err = err
}

func (i *mockPermission) GetCurrentClientVersion() (string, error) {
	return "0.0.0", i.err
}

func (i *mockPermission) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	return nil, i.err
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
func signNdf(ourNdf *pb.NDF) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	err = signature.Sign(ourNdf, ourPrivKey)
	if err != nil {
		return errors.Errorf("Could not sign ndf: %+v", err)
	}

	return nil
}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	err = signature.Sign(ri, ourPrivKey)
	if err != nil {
		return errors.Errorf("Could not sign round info: %+v", err)
	}
	return nil
}

// Utility function which creates an instance
func createServerInstance(t *testing.T) (instance *internal.Instance, pAddr,
	nodeAddr string, nodeId *id.ID, cert, key []byte, err error) {
	cert, _ = utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ = utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewIdFromUInt(uint64(0), id.Node, t)
	countLock.Lock()
	nodeAddr = fmt.Sprintf("0.0.0.0:%d", 7000+count)
	pAddr = fmt.Sprintf("0.0.0.0:%d", 2000+count)
	count++
	countLock.Unlock()
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
			TlsCert: cert,
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
		return
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err = internal.CreateServerInstance(def, impl, sm,
		"1.1.0")
	if err != nil {
		return
	}

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Permissioning.Address,
		def.Permissioning.TlsCert, params)
	if err != nil {
		return
	}
	err = nil
	return
}

// Utility function which starts up a permissioning server
func startPermissioning(pAddr, nAddr string, nodeId *id.ID, cert, key []byte, t *testing.T) (*registration.Comms, *mockPermission, error) {
	// Initialize permissioning server
	mp := &mockPermission{}
	pHandler := registration.Handler(mp)
	permComms := registration.StartRegistrationServer(&id.Permissioning, pAddr, pHandler, cert, key)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err := permComms.AddHost(nodeId, nAddr, cert, params)
	if err != nil {
		return nil, nil, errors.Errorf("Permissioning could not connect to node")
	}

	return permComms, mp, nil
}
