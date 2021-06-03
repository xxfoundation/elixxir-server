///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/grpc/metadata"
)

const testGatewayAddress = "0.0.0.0:8201"

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

// batchEq compares two batches to see if they are equal
// Return true if they are equal and false otherwise
func batchEq(a *mixmessages.Batch, b *mixmessages.Batch) bool {
	if a.GetRound() != b.GetRound() {
		return false
	}

	if a.GetFromPhase() != b.GetFromPhase() {
		return false
	}

	if len(a.GetSlots()) != len(b.GetSlots()) {
		return false
	}

	bSlots := b.GetSlots()
	for index, slot := range a.GetSlots() {
		if !reflect.DeepEqual(slot, bSlots[index]) {
			return false
		}
	}

	return true
}

func initImplGroup() *cyclic.Group {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2))
	return grp
}

// Builds a list of node IDs for testing
func buildMockTopology(numNodes int, t *testing.T) *connect.Circuit {
	var nodeIDs []*id.ID

	//Build IDs
	for i := 0; i < numNodes; i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	return connect.NewCircuit(nodeIDs)
}

// Builds a list of base64 encoded node IDs for server instance construction
func BuildMockNodeIDs(numNodes int, t *testing.T) []*id.ID {
	var nodeIDs []*id.ID

	//Build IDs
	for i := 0; i < numNodes; i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	return nodeIDs
}

func buildMockNodeAddresses(numNodes int, t *testing.T) []string {
	//generate IDs and addresses
	var nidLst []string
	var addrLst []string
	addrFmt := "localhost:5%03d"
	portState := 6000
	for i := 0; i < numNodes; i++ {
		//generate id
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nidLst = append(nidLst, nodeID.String())
		//generate address
		addr := fmt.Sprintf(addrFmt, i+portState)
		addrLst = append(addrLst, addr)
	}

	return addrLst
}

func mockServerInstance(t *testing.T, s current.Activity) (*internal.Instance, *connect.Circuit) {

	var nodeIDs []*id.ID

	for i := uint64(0); i < 3; i++ {
		nodeIDs = append(nodeIDs, id.NewIdFromUInt(i, id.Node, t))
	}

	// Generate IDs and addresses
	var nodeLst []internal.Node
	for i := 0; i < len(nodeIDs); i++ {
		// Generate id
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)

		// Generate address
		addr := fmt.Sprintf("localhost:5%03d", i)

		n := internal.Node{
			ID:      nodeID,
			Address: addr,
		}
		nodeLst = append(nodeLst, n)
	}

	topology := connect.NewCircuit(nodeIDs)
	def := internal.Definition{
		ResourceMonitor: &measure.ResourceMonitor{},
		GraphGenerator: services.NewGraphGenerator(2,
			2, 2, 0),
		RngStreamGen: fastRNG.NewStreamGenerator(10000,
			uint(runtime.NumCPU()), csprng.NewSystemRNG),
		PartialNDF: testUtil.NDF,
		FullNDF:    testUtil.NDF,
		Flags:      internal.Flags{OverrideInternalIP: "0.0.0.0"},
		DevMode:    true,
	}
	def.ID = topology.GetNodeAtIndex(0)
	def.Gateway.ID = &id.TempGateway
	def.Gateway.Address = testGatewayAddress
	m := state.NewTestMachine(dummyStates, s, t)
	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m,
		"1.1.0")

	return instance, topology
}

func mockTransmitGetMeasure(node *node.Comms, topology *connect.Circuit, roundID id.Round, t *testing.T) (string, error) {
	serverRoundMetrics := map[string]measure.RoundMetrics{}
	mockResourceMetrics := measure.ResourceMetric{
		Time:          time.Unix(int64(0), int64(1)),
		MemAllocBytes: 123,
		NumThreads:    5,
	}

	// Contact all visible servers and get metrics
	for i := 0; i < topology.Len(); i++ {
		s := topology.GetNodeAtIndex(i)

		serverRoundMetrics[s.String()] = measure.RoundMetrics{
			NodeID:         *id.NewIdFromString("NODE_TEST_ID", id.Node, t),
			RoundID:        3,
			NumNodes:       5,
			StartTime:      time.Now(),
			EndTime:        time.Now(),
			PhaseMetrics:   measure.PhaseMetrics{},
			ResourceMetric: mockResourceMetrics,
		}
	}

	ret, err := json.Marshal(serverRoundMetrics)

	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func makeMultiInstanceGroup() *cyclic.Group {
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
	return cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))
}

func PushNRoundUpdates(n int, instance internal.Instance, key *rsa.PrivateKey, t *testing.T) {

	for i := 1; i < n+1; i++ {
		newRound := &mixmessages.RoundInfo{
			ID:       uint64(i),
			UpdateID: uint64(i),
		}

		err := signature.SignRsa(newRound, key)
		if err != nil {
			t.Logf("Failed to sign: %v", err)
			t.Fail()
		}

		//t.Logf("ROUND: %v", newRound)

		err = instance.GetConsensus().RoundUpdate(newRound)
		if err != nil {
			t.Logf("error pushing round %v", err)
			t.Fail()
		}
	}

}

/*
func makeMultiInstanceParams(numNodes, batchsize, portstart int, grp *cyclic.Group) []*server.Definition {

	//generate IDs and addresses
	var nidLst []*id.ID
	var nodeLst []server.Node
	addrFmt := "localhost:5%03d"
	for i := 0; i < numNodes; i++ {
		//generate id
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nidLst = append(nidLst, nodeID)
		//generate address
		addr := fmt.Sprintf(addrFmt, i+portstart)

		n := server.Node{
			ID:      nodeID,
			ListeningAddress: addr,
		}
		nodeLst = append(nodeLst, n)

	}

	//generate parameters list
	var defLst []*server.Definition

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	for i := 0; i < numNodes; i++ {

		def := server.Definition{
			CmixGroup: grp,
			Topology:  connect.NewCircuit(nidLst),
			ID:        nidLst[i],
			BatchSize: uint32(batchsize),
			Nodes:     nodeLst,
			Flags: server.Flags{
				KeepBuffers: true,
			},
			ListeningAddress:        nodeLst[i].ListeningAddress,
			MetricsHandler: func(i *server.Instance, roundID id.Round) error { return nil },
			GraphGenerator: services.NewGraphGenerator(4, PanicHandler, 1, 4, 0.0),
			RngStreamGen: fastRNG.NewStreamGenerator(10000,
				uint(runtime.NumCPU()), csprng.NewSystemRNG),
		}

		defLst = append(defLst, &def)
	}

	return defLst
}
*/

var mockStreamSlotIndex int

/* MockStreamPostPhaseServer */
type MockStreamPostPhaseServer struct {
	batch *mixmessages.Batch
}

func (stream MockStreamPostPhaseServer) SendAndClose(*messages.Ack) error {
	if len(stream.batch.Slots) == mockStreamSlotIndex {
		return nil
	}
	return errors.New("stream closed without all slots being received")
}

func (stream MockStreamPostPhaseServer) Recv() (*mixmessages.Slots, error) {
	if mockStreamSlotIndex >= len(stream.batch.Slots) {
		return nil, io.EOF
	}
	slot := stream.batch.Slots[mockStreamSlotIndex]
	mockStreamSlotIndex++
	return &mixmessages.Slots{Messages: []*mixmessages.Slot{slot}}, nil
}

func (MockStreamPostPhaseServer) SetHeader(metadata.MD) error {
	return nil
}

func (MockStreamPostPhaseServer) SendHeader(metadata.MD) error {
	return nil
}

func (MockStreamPostPhaseServer) SetTrailer(metadata.MD) {
}

func (stream MockStreamPostPhaseServer) Context() context.Context {
	// Create mock batch info from mock batch
	mockBatch := stream.batch
	mockBatchInfo := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: mockBatch.Round.ID,
		},
		FromPhase: mockBatch.FromPhase,
		BatchSize: uint32(len(mockBatch.Slots)),
	}

	// Create an incoming context from batch info metadata
	ctx, _ := context.WithCancel(context.Background())

	m := make(map[string]string)
	m["batchinfo"] = base64.StdEncoding.EncodeToString([]byte(mockBatchInfo.String()))

	md := metadata.New(m)
	ctx = metadata.NewIncomingContext(ctx, md)

	return ctx
}

func (MockStreamPostPhaseServer) SendMsg(m interface{}) error {
	return nil
}

func (MockStreamPostPhaseServer) RecvMsg(m interface{}) error {
	return nil
}

type MockPhase struct {
	chunks  []services.Chunk
	indices []uint32
}

func (mp *MockPhase) Send(chunk services.Chunk) {
	mp.chunks = append(mp.chunks, chunk)
}

func (mp *MockPhase) Input(index uint32, slot *mixmessages.Slot) error {
	if len(slot.Salt) != 0 {
		return errors.New("error to test edge case")
	}
	mp.indices = append(mp.indices, index)
	return nil
}

func (*MockPhase) EnableVerification() { return }
func (*MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	return
}
func (*MockPhase) GetGraph() *services.Graph { return nil }
func (*MockPhase) GetRoundID() id.Round      { return 0 }
func (*MockPhase) GetType() phase.Type       { return 0 }
func (*MockPhase) GetState() phase.State     { return 0 }
func (mp *MockPhase) AttemptToQueue(queue chan<- phase.Phase) bool {
	queue <- mp
	return true
}
func (mp *MockPhase) IsQueued() bool                      { return true }
func (*MockPhase) UpdateFinalStates()                     { return }
func (*MockPhase) GetTransmissionHandler() phase.Transmit { return nil }
func (*MockPhase) GetTimeout() time.Duration              { return 0 }
func (*MockPhase) Cmp(phase.Phase) bool                   { return false }
func (*MockPhase) String() string                         { return "" }
func (*MockPhase) Measure(string)                         { return }
func (*MockPhase) GetMeasure() measure.Metrics            { return *new(measure.Metrics) }
func (*MockPhase) GetAlternate() (bool, func())           { return false, nil }

func buildTestNetworkComponents(impls []*node.Implementation, portStart int,
	t *testing.T) ([]*node.Comms, *connect.Circuit) {
	var nodeIDs []*id.ID
	var addrLst []string
	addrFmt := "localhost:3%03d"

	//Build IDs and addresses
	for i := 0; i < len(impls); i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
		addrLst = append(addrLst, fmt.Sprintf(addrFmt, i+portStart))
	}

	//Build the topology
	topology := connect.NewCircuit(nodeIDs)

	//build the comms
	var comms []*node.Comms

	for index, impl := range impls {
		nodeID := id.NewIdFromUInt(uint64(index), id.Node, t)
		comms = append(comms,
			node.StartNode(nodeID, addrLst[index], 0, impl, nil, nil))
	}

	//Connect the comms
	for connectFrom := 0; connectFrom < len(impls); connectFrom++ {
		for connectTo := 0; connectTo < len(impls); connectTo++ {
			params := connect.GetDefaultHostParams()
			params.AuthEnabled = false
			tmpHost, _ := comms[connectFrom].AddHost(topology.GetNodeAtIndex(connectTo),
				addrLst[connectTo], nil, params)
			topology.AddHost(tmpHost)
		}
	}

	//Return comms and topology
	return comms, topology
}

func Shutdown(comms []*node.Comms) {
	for _, comm := range comms {
		comm.Shutdown()
	}
}
