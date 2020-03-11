package permissioning

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"time"
)

var nodeId *id.Node
var permComms *registration.Comms
var gwComms *gateway.Comms
var testNdf *ndf.NetworkDefinition

// Dummy implementation of permissioning server --------------------------------
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

	return &pb.PermissionPollResponse{
		FullNDF: &pb.NDF{
			Ndf:       fullNdf,
			Signature: &pb.RSASignature{Signature: testUtil.SignatureBytes},
		},
		PartialNDF: &pb.NDF{
			Ndf:       stripNdf,
			Signature: nil,
		},
	}, nil
}

func (i *mockPermission) GetCurrentClientVersion() (string, error) {
	return "0.0.0", nil
}

func (i *mockPermission) GetUpdatedNDF(clientNDFHash []byte) ([]byte, error) {
	return nil, nil
}

// Dummy implementation of gateway server --------------------------------------
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
