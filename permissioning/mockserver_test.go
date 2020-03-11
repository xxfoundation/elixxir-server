package permissioning

import (
	"fmt"
	"github.com/pkg/errors"
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
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/node/receivers"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"math/rand"
	"testing"
	"time"
)

var nodeId *id.Node
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

func (i *mockPermission) RegisterNode([]byte, string, string, string, string, string) error {
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

// --------------------------Dummy implementation of gateway server --------------------------------------
type mockGateway struct{}

func (*mockGateway) CheckMessages(userID *id.User, messageID string, ipAddress string) ([]string, error) {
	return nil, nil
}

func (*mockGateway) GetMessage(userID *id.User, msgID string, ipAddress string) (*pb.Slot, error) {
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

func (*mockGateway) PollForNotifications(auth *connect.Auth) ([]string, error) {
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

func mockServerDef(i interface{}) *server.Definition {
	nid := server.GenerateId(i)

	resourceMetric := measure.ResourceMetric{
		Time:          time.Now(),
		MemAllocBytes: 0,
		NumThreads:    0,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&resourceMetric)

	def := server.Definition{
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

func buildMockNdf(nodeId *id.Node, nodeAddress, gwAddress string, cert, key []byte) {
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
func signRoundInfo(t *testing.T, ri *pb.RoundInfo) {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)

}

// Utility function which builds a signed full-ndf message
func setupFullNdf(t *testing.T) *mixmessages.NDF {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}
	tmpNdf, _, _ := ndf.DecodeNDF(testUtil.ExampleJSON)
	f.Ndf, err = tmpNdf.Marshal()
	if err != nil {
		t.Errorf("Failed to marshal ndf: %+v", err)
	}

	if err != nil {
		t.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f
}

// Utility function which builds a signed partial-ndf message
func setupPartialNdf(t *testing.T) *mixmessages.NDF {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	f := &mixmessages.NDF{}

	stipped, err := testUtil.NDF.StripNdf().Marshal()
	f.Ndf = stipped

	if err != nil {
		t.Errorf("Could not generate serialized ndf: %s", err)
	}

	err = signature.Sign(f, ourPrivKey)

	return f
}

// Utility function which creates an instance
func createServerInstance(t *testing.T) (*server.Instance, error) {
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	nodeAddr = fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000)+cnt)
	pAddr = fmt.Sprintf("0.0.0.0:%d", 2000+rand.Intn(1000))
	cnt++
	// Build the node
	emptyNdf := builEmptydMockNdf()
	// Initialize definition
	def := &server.Definition{
		Flags:         server.Flags{},
		ID:            nodeId,
		PublicKey:     nil,
		PrivateKey:    nil,
		TlsCert:       cert,
		TlsKey:        key,
		Address:       nodeAddr,
		LogPath:       "",
		MetricLogPath: "",
		UserRegistry:  nil,
		Permissioning: server.Perm{
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
	impl := func(i *server.Instance) *node.Implementation {
		return receivers.NewImplementation(i)
	}

	// Generate instance
	instance, err := server.CreateServerInstance(def, impl, sm, true)
	if err != nil {
		return nil, errors.Errorf("Unable to create instance: %+v", err)
	}

	// Add permissioning as a host
	_, err = instance.GetNetwork().AddHost(id.PERMISSIONING, def.Permissioning.Address,
		def.Permissioning.TlsCert, false, false)
	if err != nil {
		return nil, errors.Errorf("Failed to add permissioning host: %+v", err)
	}

	return instance, nil
}

// Utility function which starts up a permissioning server
func startPermisioning() (*registration.Comms, error) {

	cert := []byte(testUtil.RegCert)
	key := []byte(testUtil.RegPrivKey)
	// Initialize permissioning server
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer(id.PERMISSIONING, pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId.String(), pAddr, cert, false, false)
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
	gwComms = gateway.StartGateway(nodeId.NewGateway().String(), gAddr, gHandler, cert, key)
	_, err := gwComms.AddHost(nodeId.String(), nodeAddr, cert, false, false)
	if err != nil {
		return nil, err
	}

	return gwComms, nil
}
