////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"os"
	"runtime"
	"testing"
	"time"
)

func setup(t *testing.T) (*internal.Instance, *connect.Circuit) {
	var nodeIDs []*id.ID

	//Build IDs
	for i := 0; i < 5; i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	topology := connect.NewCircuit(nodeIDs)
	gg := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)
	def := internal.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		GraphGenerator:  gg,
		Gateway: internal.GW{
			Address: "0.0.0.0:11420",
		},
		Address: "0.0.0.0:11421",
	}
	def.ID = topology.GetNodeAtIndex(0)

	var instance *internal.Instance
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
	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)
	instance, _ = internal.CreateServerInstance(&def, io.NewImplementation,
		m, false, "1.1.0", "/tmp/recovered_error")

	_, err := instance.GetNetwork().AddHost(&id.Permissioning, testUtil.NDF.Registration.Address,
		[]byte(testUtil.RegCert), false, false)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}
	r := round.NewDummyRoundWithTopology(id.Round(0), 3, topology, t)
	instance.GetRoundManager().AddRound(r)
	_ = instance.Run()
	return instance, topology
}

func TestNewStateChanges(t *testing.T) {
	ourStates := NewStateChanges()
	if len(ourStates) != int(current.NUM_STATES) {
		t.Errorf("Length of state table is not of expected length: "+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", int(current.NUM_STATES), ourStates)
	}

	for i := 0; i < int(current.NUM_STATES); i++ {
		if ourStates[i] == nil {
			t.Errorf("Case %d wasn't initialized, should not be nil!", i)
		}

	}
}

/*func TestNotStarted_RoundError(t *testing.T) {
	instance, _ := setup(t)
	err := NotStarted(instance, true)
	if err != nil {
		t.Error(err)
	}
}*/

func TestError(t *testing.T) {
	instance, topology := setup(t)
	rndErr := &mixmessages.RoundError{
		Id:     0,
		NodeId: instance.GetID().Marshal(),
		Error:  "",
	}
	mockBroadcast := func(host *connect.Host, message *mixmessages.RoundError) (*mixmessages.Ack, error) {
		return nil, nil
	}
	instance.SetRoundErrFunc(mockBroadcast, t)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in SetGroup(): ", r)
		} else {
			t.Errorf("SetGroup() did not panic when expected while attempting to set the group again")
		}
		instance.GetNetwork().Shutdown()
	}()

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	instance.SetTestRoundError(rndErr, t)

	err := Error(instance)
	if err != nil {
		t.Errorf("Failed to error: %+v", err)
	}
}

func TestPrecomputing(t *testing.T) {
	var err error
	instance, topology := setup(t)

	var top [][]byte
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		top = append(top, nid.Marshal())
		_, err = instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	newRoundInfo := &mixmessages.RoundInfo{
		ID:        0,
		Topology:  top,
		BatchSize: 32,
	}

	// Mocking permissioning server signing message
	err = signRoundInfo(newRoundInfo)
	if err != nil {
		t.Errorf("failed to sign round info")
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = instance.GetCreateRoundQueue().Send(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to send roundInfo: %v", err)
	}

	err = instance.GetResourceQueue().Kill(time.Millisecond * 10)
	if err != nil {
		t.Errorf("Failed to kill resource queue: %+v", err)
	}

	err = Precomputing(instance, 3)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	_, err = instance.GetRoundManager().GetRound(0)
	if err != nil {
		t.Errorf("A round should have been added to the round manager")
	}
	instance.GetNetwork().Shutdown()
}

func TestPrecomputing_override(t *testing.T) {
	var err error
	instance, topology := setup(t)
	gc := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 0)
	g := graphs.InitErrorGraph(gc)
	th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return errors.New("Failed intentionally")
	}
	overrides := map[int]phase.Phase{}
	p := phase.New(phase.Definition{
		Graph:               g,
		Type:                phase.PrecompGeneration,
		TransmissionHandler: th,
		Timeout:             30127,
		DoVerification:      false,
	})
	overrides[0] = p
	instance.OverridePhasesAtRound(overrides, 1)

	var top [][]byte
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		top = append(top, nid.Marshal())
		_, err = instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	newRoundInfo := &mixmessages.RoundInfo{
		ID:        1,
		Topology:  top,
		BatchSize: 32,
	}
	// Mocking permissioning server signing message
	err = signRoundInfo(newRoundInfo)
	if err != nil {
		t.Errorf("failed to sign round info")
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = instance.GetCreateRoundQueue().Send(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to send roundInfo: %v", err)
	}

	err = instance.GetResourceQueue().Kill(time.Millisecond * 10)
	if err != nil {
		t.Errorf("Failed to kill resource queue: %+v", err)
	}

	err = Precomputing(instance, 3)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	rnd, _ := instance.GetRoundManager().GetRound(id.Round(1))
	phase, _ := rnd.GetPhase(phase.PrecompGeneration)
	if phase.GetTimeout() != 30127 {
		t.Error("Failed to override phase")
	}
}

// Utility function which signs a round info message
func signRoundInfo(ri *mixmessages.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)
	return nil
}

// Tests that getCertificates() correctly finds and retrieves Server and Gateway
// certificates from file.
func Test_getCertificates(t *testing.T) {
	// Set up server definition with Server and Gateway certificate paths
	def := &internal.Definition{
		ServerCertPath:  "server-temp.cert",
		GatewayCertPath: "gateway-temp.cert",
	}

	serverCertData := []byte("Test Server string.")
	gatewayCertData := []byte("Test Gateway string.")

	// Create temporary certificate files for testing
	err := utils.WriteFile(def.ServerCertPath, serverCertData, utils.FilePerms,
		utils.FilePerms)
	if err != nil {
		t.Errorf("WriteFile() produced an unexpected error creating "+
			"%v:\n\t%v", def.ServerCertPath, err)
	}

	err = utils.WriteFile(def.GatewayCertPath, gatewayCertData, utils.FilePerms,
		utils.FilePerms)
	if err != nil {
		t.Errorf("WriteFile() produced an unexpected error creating "+
			"%v:\n\t%v", def.GatewayCertPath, err)
	}

	// Attempt to get certificates
	certsExist, serverCert, gatewayCert := getCertificates(def.ServerCertPath,
		def.GatewayCertPath)

	// Check that the certificates were found
	if !certsExist {
		t.Errorf("getCertificates() did not find one or both certificates " +
			"that should exist.")
	}

	// Check that the retrieved Server certificate has the correct contents
	if serverCert != string(serverCertData) {
		t.Errorf("getCertificates() returned an unexpected Server certificate."+
			"\n\texpected: %s\n\treceived: %s",
			string(serverCertData), serverCert)
	}

	// Check that the retrieved Gateway certificate has the correct contents
	if gatewayCert != string(gatewayCertData) {
		t.Errorf("getCertificates() returned an unexpected Gateway certificate."+
			"\n\texpected: %s\n\treceived: %s",
			string(gatewayCertData), gatewayCert)
	}

	// Clean up test files
	_ = os.RemoveAll(def.ServerCertPath)
	_ = os.RemoveAll(def.GatewayCertPath)
}

// Tests that writeCertificates() panics when the Gateway certificate cannot be
// written to file.
func Test_getCertificates_PanicGateway(t *testing.T) {
	// Set up server definition with Server and Gateway certificate paths; the
	// Gateway path is invalid.
	def := &internal.Definition{
		ServerCertPath:  "server-temp.cert",
		GatewayCertPath: "~a/gateway-temp.cert",
	}

	serverCertData := "Test Server string."
	gatewayCertData := "Test Gateway string."

	// Defer to an error when writeCertificates() does not panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("writeCertificates() did not panic when expected")
		}

		// Clean up test files
		_ = os.RemoveAll(def.ServerCertPath)
		_ = os.RemoveAll(def.GatewayCertPath)
	}()

	// Attempt to write certificates
	writeCertificates(def, serverCertData, gatewayCertData)
}

// Tests that writeCertificates() correctly saves the Server and Gateway
// certificates to file.
func Test_writeCertificates(t *testing.T) {
	// Set up server definition with Server and Gateway certificate paths
	def := &internal.Definition{
		ServerCertPath:  "server-temp.cert",
		GatewayCertPath: "gateway-temp.cert",
	}

	serverCertData := "Test Server string."
	gatewayCertData := "Test Gateway string."

	// Attempt to write certificates
	writeCertificates(def, serverCertData, gatewayCertData)

	// Attempt to get certificates
	certsExist, serverCert, gatewayCert := getCertificates(def.ServerCertPath,
		def.GatewayCertPath)

	// Check that the certificates were both written
	if !certsExist {
		t.Errorf("writeCertificates() did not write one or both " +
			"certificates to file.")
	}

	// Check that the Server certificate was correctly written to file
	if serverCert != serverCertData {
		t.Errorf("writeCertificates() wrote the wrong Server certificate "+
			"data to file.\n\texpected: %s\n\treceived: %s",
			serverCertData, serverCert)
	}

	// Check that the Gateway certificate was correctly written to file
	if gatewayCert != gatewayCertData {
		t.Errorf("writeCertificates() wrote the wrong Gateway certificate "+
			"data to file.\n\texpected: %s\n\treceived: %s",
			gatewayCertData, gatewayCert)
	}

	// Clean up test files
	_ = os.RemoveAll(def.ServerCertPath)
	_ = os.RemoveAll(def.GatewayCertPath)
}

// Tests that writeCertificates() panics when the Server certificate cannot be
// written to file.
func Test_writeCertificates_PanicServer(t *testing.T) {
	// Set up server definition with Server and Gateway certificate paths; the
	// Server path is invalid.
	def := &internal.Definition{
		ServerCertPath:  "~a/server-temp.cert",
		GatewayCertPath: "gateway-temp.cert",
	}

	serverCertData := "Test Server string."
	gatewayCertData := "Test Gateway string."

	// Defer to an error when writeCertificates() does not panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("writeCertificates() did not panic when expected")
		}

		// Clean up test files
		_ = os.RemoveAll(def.ServerCertPath)
		_ = os.RemoveAll(def.GatewayCertPath)
	}()

	// Attempt to write certificates
	writeCertificates(def, serverCertData, gatewayCertData)
}

// Tests that writeCertificates() panics when the Gateway certificate cannot be
// written to file.
func Test_writeCertificates_PanicGateway(t *testing.T) {
	// Set up server definition with Server and Gateway certificate paths; the
	// Gateway path is invalid.
	def := &internal.Definition{
		ServerCertPath:  "server-temp.cert",
		GatewayCertPath: "~a/gateway-temp.cert",
	}

	serverCertData := "Test Server string."
	gatewayCertData := "Test Gateway string."

	// Defer to an error when writeCertificates() does not panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("writeCertificates() did not panic when expected")
		}

		// Clean up test files
		_ = os.RemoveAll(def.ServerCertPath)
		_ = os.RemoveAll(def.GatewayCertPath)
	}()

	// Attempt to write certificates
	writeCertificates(def, serverCertData, gatewayCertData)
}
