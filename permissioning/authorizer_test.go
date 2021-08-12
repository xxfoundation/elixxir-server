///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"fmt"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"runtime"
	"testing"
)

// Happy path
func TestAuthorize(t *testing.T) {

	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Fatalf("Failed to read cert file: %+v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Failed to read key file: %+v", err)
	}

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)

	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7100+count)
	authAddr := fmt.Sprintf("0.0.0.0:%d", 2100+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4100+count)
	count++
	countLock.Unlock()

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	//make server rsa key pair
	pk, _ := utils.ReadFile(testkeys.GetNodeKeyPath())
	privKey, _ := rsa.LoadPrivateKeyFromPem(pk)

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        privKey.GetPublic(),
		PrivateKey:       privKey,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
		Network: internal.Perm{
			TlsCert: cert,
			Address: authAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
		DevMode:         true,
		RngStreamGen: fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG),
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Fatalf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Fatalf("Unable to create instance: %+v", err)
	}

	// Upsert test data for gateway
	instance.UpsertGatewayData("0.0.0.0:5289", "1.4")

	// Start up permissioning server
	authComms, err := startAuthorizer(authAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Fatalf("Couldn't create permissioning server: %+v", err)
	}
	defer authComms.Shutdown()

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	authHost, err := instance.GetNetwork().AddHost(&id.Authorizer, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	err = Authorize(instance, authHost)
	if err != nil {
		t.Fatalf("Could not authorize: %v", err)
	}
}

// Error path
func TestAuthorize_Error(t *testing.T) {
	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Fatalf("Failed to read cert file: %+v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Failed to read key file: %+v", err)
	}

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)

	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7100+count)
	authAddr := fmt.Sprintf("0.0.0.0:%d", 2100+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4100+count)
	count++
	countLock.Unlock()

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	//make server rsa key pair
	pk, _ := utils.ReadFile(testkeys.GetNodeKeyPath())
	privKey, _ := rsa.LoadPrivateKeyFromPem(pk)

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        privKey.GetPublic(),
		PrivateKey:       privKey,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
		Network: internal.Perm{
			TlsCert: cert,
			Address: authAddr,
		},
		RegistrationCode: "",

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
		DevMode:         true,
		RngStreamGen: fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG),
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Fatalf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Fatalf("Unable to create instance: %+v", err)
	}

	// Upsert test data for gateway
	instance.UpsertGatewayData("0.0.0.0:5289", "1.4")

	// Start up permissioning server
	authComms, err := startAuthorizerErrorPath(authAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Fatalf("Couldn't create permissioning server: %+v", err)
	}
	defer authComms.Shutdown()

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	authHost, err := instance.GetNetwork().AddHost(&id.Authorizer, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	err = Authorize(instance, authHost)
	if err == nil {
		t.Fatalf("Expected error path did not return an error: %v", err)
	}
}

// Happy path
func TestSend(t *testing.T) {
	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Fatalf("Failed to read cert file: %+v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Failed to read key file: %+v", err)
	}

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)

	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7100+count)
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2100+count)
	authAddr := fmt.Sprintf("0.0.0.0:%d", 2200+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4100+count)
	count++
	countLock.Unlock()

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	//make server rsa key pair
	pk, _ := utils.ReadFile(testkeys.GetNodeKeyPath())
	privKey, _ := rsa.LoadPrivateKeyFromPem(pk)

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        privKey.GetPublic(),
		PrivateKey:       privKey,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
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
		RngStreamGen: fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG),
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Fatalf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Fatalf("Unable to create instance: %+v", err)
	}

	// Start up permissioning server
	authComms, err := startAuthorizer(authAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Fatalf("Couldn't create permissioning server: %+v", err)
	}
	defer authComms.Shutdown()

	// Start up permissioning server
	permComms, err := startPermissioning(pAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Authorizer, authAddr,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Add permissioning as a host
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Construct sender interface
	sendFunc := func(host *connect.Host) (interface{}, error) {
		pollMsg := &pb.PermissioningPoll{}
		return instance.GetNetwork().SendPoll(host, pollMsg)

	}

	sender := Sender{
		Send: sendFunc,
		Name: "Test",
	}

	response, err := Send(sender, instance, nil)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	_, ok = response.(*pb.PermissionPollResponse)
	if !ok {
		t.Fatalf("Unexpected response: " +
			"Could not cast to expected message (PermissionPollResponse)")
	}
}

// Error path: Simulate connection refused by permissioning,
// followed by a successful authorization and successful resend
func TestSend_ErrorOnce(t *testing.T) {
	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Fatalf("Failed to read cert file: %+v", err)
	}
	key, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Failed to read key file: %+v", err)
	}

	// Set up id's and address
	nodeId := id.NewIdFromUInt(0, id.Node, t)

	countLock.Lock()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7100+count)
	pAddr := fmt.Sprintf("0.0.0.0:%d", 2100+count)
	authAddr := fmt.Sprintf("0.0.0.0:%d", 2200+count)
	gAddr := fmt.Sprintf("0.0.0.0:%d", 4100+count)
	count++
	countLock.Unlock()

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	// Build the node
	emptyNdf := builEmptydMockNdf()

	//make server rsa key pair
	pk, _ := utils.ReadFile(testkeys.GetNodeKeyPath())
	privKey, _ := rsa.LoadPrivateKeyFromPem(pk)

	// Initialize definition
	def := &internal.Definition{
		Flags:            internal.Flags{},
		ID:               nodeId,
		PublicKey:        privKey.GetPublic(),
		PrivateKey:       privKey,
		TlsCert:          cert,
		TlsKey:           key,
		ListeningAddress: nodeAddr,
		PublicAddress:    nodeAddr,
		LogPath:          "",
		Gateway: internal.GW{
			ID:      gwID,
			Address: gAddr,
			TlsCert: cert,
		},
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
		RngStreamGen: fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG),
	}

	// Create state machine
	sm := state.NewMachine(dummyStates)
	ok, err := sm.Update(current.WAITING)
	if !ok || err != nil {
		t.Fatalf("Failed to prep state machine: %+v", err)
	}

	// Add handler for instance
	impl := func(i *internal.Instance) *node.Implementation {
		return io.NewImplementation(i)
	}

	// Generate instance
	instance, err := internal.CreateServerInstance(def, impl, sm, "1.1.0")
	if err != nil {
		t.Fatalf("Unable to create instance: %+v", err)
	}

	// Start up permissioning server
	authComms, err := startAuthorizer(authAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Fatalf("Couldn't create permissioning server: %+v", err)
	}
	defer authComms.Shutdown()

	// Start up permissioning server
	permComms, err := startPermissioning_ConnectionErrorOnce(pAddr, nodeAddr, nodeId, cert, key)
	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Add permissioning as a host
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = instance.GetNetwork().AddHost(&id.Authorizer, authAddr,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Add permissioning as a host
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, def.Network.Address,
		def.Network.TlsCert, params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}

	// Send message
	sendFunc := func(host *connect.Host) (interface{}, error) {
		registrationRequest := &pb.NodeRegistration{
			Salt:             def.Salt,
			ServerTlsCert:    string(def.TlsCert),
			GatewayTlsCert:   string(def.Gateway.TlsCert),
			ServerAddress:    nodeAddr,
			RegistrationCode: def.RegistrationCode,
		}

		return nil, instance.GetNetwork().SendNodeRegistration(host, registrationRequest)

	}

	sender := Sender{
		Send: sendFunc,
		Name: "Test",
	}

	_, err = Send(sender, instance, nil)
	if err != nil {
		t.Fatalf("Expected happy path: Should have authorized "+
			"and returned no error. Returned error: %v", err)
	}

}
