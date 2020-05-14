package permissioning

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
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

func (i *mockPermission) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
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

func (i *mockPermissionMultipleRounds) Poll(*pb.PermissioningPoll, *connect.Auth) (*pb.PermissionPollResponse, error) {
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

	// Create a topology for round info
	jww.FATAL.Println(node1)
	ourTopology := [][]byte{node1, node2, node3}

	// Construct round info message indicating PRECOMP starting
	precompRoundInfo := &pb.RoundInfo{
		ID:       0,
		UpdateID: numUpdates,
		State:    uint32(states.PRECOMPUTING),
		Topology: ourTopology,
	}

	// Mocking permissioning server signing message
	signRoundInfo(precompRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Construct round info message indicating STANDBY starting
	standbyRoundInfo := &pb.RoundInfo{
		ID:       0,
		UpdateID: numUpdates,
		State:    uint32(states.STANDBY),
		Topology: ourTopology,
	}

	// Mocking permissioning server signing message
	signRoundInfo(standbyRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Construct message which adds node to team
	ourTopology = append(ourTopology, node4)

	// Add new round in standby stage
	newNodeRoundInfo := &pb.RoundInfo{
		ID:       0,
		UpdateID: numUpdates,
		State:    uint32(states.STANDBY),
		Topology: ourTopology,
	}

	// Set the signature field of the round info
	signRoundInfo(newNodeRoundInfo)

	// Increment updates id for next message
	numUpdates++

	// Create a time stamp in which to transfer stats
	ourTime := time.Now().Add(500 * time.Millisecond).UnixNano()
	timestamps := make([]uint64, states.FAILED)
	timestamps[states.REALTIME] = uint64(ourTime)

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

	cmixGroup := ndf.Group{
		Prime:      "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		SmallPrime: "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		Generator:  "02",
	}

	e2eGroup := ndf.Group{
		Prime:      "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		SmallPrime: "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		Generator:  "02",
	}

	ourMockNdf := &ndf.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{},
		Gateways:  []ndf.Gateway{},
		E2E:       e2eGroup,
		CMIX:      cmixGroup,
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
			TlsCert:          []byte(testUtil.RegCert),
			Address:          pAddr,
			RegistrationCode: "",
		},

		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		FullNDF:         emptyNdf,
		PartialNDF:      emptyNdf,
	}

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
	instance, err := internal.CreateServerInstance(def, impl, sm, true)
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
