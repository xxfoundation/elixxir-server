package io

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

// Mock an implementation with a GetMeasure function.
func MockRTPingImplementation() *node.Implementation {
	impl := node.NewImplementation()

	impl.Functions.SendRoundTripPing = func(ping *mixmessages.RoundTripPing) error {
		return nil
	}

	return impl
}

func TestTransmitRoundTripPing(t *testing.T) {
	// Setup the network
	impl := MockRTPingImplementation()

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10)
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

	nid := server.GenerateId()
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	def := server.Definition{
		CmixGroup: grp,
		Nodes: []server.Node{
			{
				ID: nid,
			},
		},
		ID:              nid,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		PrivateKey:      mockRSAPriv,
		PublicKey:       mockRSAPub,
	}

	mockServerInstance := server.CreateServerInstance(&def)
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

	r := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		mockServerInstance.GetRngStreamGen(), "0.0.0.0")

	before := r.GetRTStart().String()

	err = TransmitRoundTripPing(comms[0], topology.GetNodeAtIndex(1),
		r, &mixmessages.Ack{}, "EMPTY/ACK")
	if err != nil {
		t.Errorf("Error transmitting rt ping: %+v", err)
	}

	if before == r.GetRTStart().String() {
		t.Error("RT Start time did not change")
	}
}