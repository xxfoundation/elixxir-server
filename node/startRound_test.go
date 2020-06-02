package node

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/io"

	//"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
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

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
			t.Fail()
		}
	}()
	f()
}

func setupStartNode(t *testing.T) *internal.Instance {
	//Get a new ndf
	testNdf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil {
		t.Logf("Failed to decode ndf")
		t.Fail()
	}

	// We need to create a server.Definition so we can create a server instance.
	def := internal.Definition{
		ID:              id.NewIdFromUInt(0, id.Node, t),
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testNdf,
		PartialNDF:      testNdf,
	}

	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)

	instance, err := internal.CreateServerInstance(&def, io.NewImplementation, m, false, "1.1.0")
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	// In order for our instance to return updated ndf we need to sign it so here we extract keys
	cert := testUtil.RegCert

	if err != nil {
		t.Logf("Private Key failed to generate %v", err)
		t.Fail()
	}

	// Add the certs to our network instance
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, "", []byte(cert), false, false)
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}

	return instance
}

func createRound(roundId id.Round, instance *internal.Instance, t *testing.T) *round.Round {

	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"

	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	mockPhase := testUtil.InitMockPhase(t)
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhase.GetType()},
	)

	batchSize := uint32(10)

	list := []*id.ID{}

	for i := uint64(0); i < 8; i++ {
		node := id.NewIdFromUInt(i, id.Node, t)
		list = append(list, node)
	}

	top := connect.NewCircuit(list)

	r, err := round.New(grp, &globals.UserMap{}, roundId, []phase.Phase{mockPhase},
		responseMap, top, top.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil)

	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
		t.FailNow()
	}

	return r
}

func TestStartLocalPrecomp_HappyPath(t *testing.T) {
	instance := setupStartNode(t)
	roundId := id.Round(0)
	r := createRound(roundId, instance, t)
	instance.GetRoundManager().AddRound(r)

	newRoundInfo := &mixmessages.RoundInfo{
		ID:        0,
		Topology:  [][]byte{instance.GetID().Marshal()},
		BatchSize: 32,
	}

	// Mocking permissioning server signing message
	signRoundInfo(newRoundInfo)

	err := instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = StartLocalPrecomp(instance, roundId)
	if err != nil {
		t.Logf("%v", err)
		t.Fail()
	}
}

// Test that if there is no round we catch a panic
func TestStartLocalPrecomp_NoRoundError(t *testing.T) {
	instance := setupStartNode(t)
	roundId := id.Round(0)

	assertPanic(t, func() {
		err := StartLocalPrecomp(instance, roundId)
		if err != nil {
			t.Logf("%v", err)
			t.Fail()
		}
	})
}
