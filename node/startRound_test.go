package node

import (
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/server/server/phase"

	//"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node/receivers"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

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


func setupStartNode(t *testing.T) (*server.Instance, id.Round){
	//Get a new ndf
	testNdf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil {
		t.Logf("Failed to decode ndf")
		t.Fail()
	}

	// We need to create a server.Definition so we can create a server instance.
	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testNdf,
		PartialNDF:      testNdf,
	}


	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)

	instance, err := server.CreateServerInstance(&def, receivers.NewImplementation, m, false)
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	//topology := connect.NewCircuit(receivers.BuildMockNodeIDs(10))

	// In order for our instance to return updated ndf we need to sign it so here we extract keys
	certPath := testkeys.GetGatewayCertPath()
	cert := testkeys.LoadFromPath(certPath)
	//keyPath := testkeys.GetGatewayKeyPath()
	//privKeyPem := testkeys.LoadFromPath(keyPath)
	//privKey, err := rsa.LoadPrivateKeyFromPem(privKeyPem)
	if err != nil {
		t.Logf("Private Key failed to generate %v", err)
		t.Fail()
	}

	// Add the certs to our network instance
	_, err = instance.GetNetwork().AddHost(id.PERMISSIONING, "", cert, false, false)
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}
	roundId := id.Round(23)

	phaseDef := phase.Definition{
		Graph:               nil,
		Type:                1,
		TransmissionHandler: nil,
		Timeout:             0,
		DoVerification:      false,
	}

	newPhase := phase.New(phaseDef)
	dr := round.NewDummyRound(roundId, 10,[]phase.Phase{newPhase},t)
	instance.GetRoundManager().AddRound(dr)

	return  instance, roundId
}


func TestStartLocalPrecomp_HappyPath(t *testing.T) {

	instance, roundId := setupStartNode(t)
	err := StartLocalPrecomp(instance, roundId)
	if err != nil {
		t.Logf("%v",err)
		t.Fail()
	}

}

// Test that if there is no round we catch a panic
func TestStartLocalPrecomp_NoRoundError(t *testing.T) {
	instance, roundId := setupStartNode(t)
	err := StartLocalPrecomp(instance, roundId)
	if err != nil {
		t.Logf("%v",err)
		t.Fail()
	}
}

// Test that if there is no round we catch a panic
func TestStartLocalPrecomp_PostPhaseFail(t *testing.T) {
	instance, roundId := setupStartNode(t)
	err := StartLocalPrecomp(instance, roundId)
	if err != nil {
		t.Logf("%v",err)
		t.Fail()
	}
}
