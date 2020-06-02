package io

import (
	"fmt"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

// Mock an implementation with a GetMeasure function.
func MockRTPingImplementation() *node.Implementation {
	impl := node.NewImplementation()

	impl.Functions.SendRoundTripPing = func(ping *mixmessages.RoundTripPing, auth *connect.Auth) error {
		return nil
	}

	return impl
}

func TestTransmitRoundTripPing(t *testing.T) {
	// Setup the network
	impl := MockRTPingImplementation()

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10, t)
	defer Shutdown(comms)

	mockRSAPriv, err := rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate node private key: %+v", err))
	}

	mockRSAPub := mockRSAPriv.GetPublic()

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

	nid := internal.GenerateId(t)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	def := internal.Definition{
		ID:              nid,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		PrivateKey:      mockRSAPriv,
		PublicKey:       mockRSAPub,
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
	}
	nodeIDs := make([]*id.ID, 0)
	nodeIDs = append(nodeIDs, nid)

	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)
	mockServerInstance, _ := internal.CreateServerInstance(&def, NewImplementation, m, false, "1.1.0")
	mockServerInstance.GetNetwork()

	roundID := id.Round(0)

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

	batchSize := uint32(11)

	r, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		mockServerInstance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	before := r.GetRTStart().String()

	err = TransmitRoundTripPing(comms[0], topology.GetNodeAtIndex(1),
		r, &mixmessages.Ack{}, "EMPTY/ACK", nil)
	if err != nil {
		t.Errorf("Error transmitting rt ping: %+v", err)
	}

	if before == r.GetRTStart().String() {
		t.Error("RT Start time did not change")
	}
}
