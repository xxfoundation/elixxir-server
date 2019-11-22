////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/services"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var nodeId *id.Node
var permComms *registration.Comms
var gwComms *gateway.Comms

// Dummy implementation of permissioning server --------------------------------
type mockPermission struct{}

func (i *mockPermission) RegisterUser(registrationCode, test string) (hash []byte, err error) {
	return nil, nil
}

func (i *mockPermission) RegisterNode(ID []byte, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

	go func() {
		nodeTop := make([]*pb.NodeInfo, 0)
		nodeTop = append(nodeTop, &pb.NodeInfo{
			Id:             nodeId.Bytes(),
			Index:          0,
			ServerAddress:  ServerAddr,
			ServerTlsCert:  "a",
			GatewayTlsCert: "b",
			GatewayAddress: GatewayAddr,
		})
		nwTop := &pb.NodeTopology{
			Topology: nodeTop,
		}

		nodeHost, _ := permComms.GetHost(nodeId.String())

		err := permComms.SendNodeTopology(nodeHost, nwTop)
		if err != nil {
			panic(err)
		}
	}()

	return nil
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

// -----------------------------------------------------------------------------

// Full-stack happy path test for the node registration logic
func TestRegisterNode(t *testing.T) {

	gwConnected := make(chan struct{})
	permDone := make(chan struct{})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	addr := fmt.Sprintf("0.0.0.0:%d", 6000+rand.Intn(1000))

	// Initialize permissioning server
	pAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer(pAddr, pHandler, cert, key)
	_, err := permComms.AddHost(nodeId.String(), addr, cert, false)
	if err != nil {
		t.Fatalf("Permissioning could not connect to node")
	}

	gAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	gHandler := gateway.Handler(&mockGateway{})
	gwComms = gateway.StartGateway(gAddr, gHandler, cert, key)

	go func() {
		time.Sleep(1 * time.Second)
		gwComms.AddHost(nodeId.String(), addr, cert, false)
		if err != nil {
			t.Fatalf("Gateway could not connect to node")
		}
		gwConnected <- struct{}{}
	}()

	// Initialize definition
	def := &server.Definition{
		Flags:         server.Flags{},
		ID:            nodeId,
		PublicKey:     nil,
		PrivateKey:    nil,
		TlsCert:       cert,
		TlsKey:        key,
		Address:       addr,
		LogPath:       "",
		MetricLogPath: "",
		Gateway: server.GW{
			Address: gAddr,
			TlsCert: cert,
		},
		UserRegistry:    nil,
		GraphGenerator:  services.GraphGenerator{},
		ResourceMonitor: nil,
		BatchSize:       0,
		CmixGroup:       nil,
		E2EGroup:        nil,
		Topology:        nil,
		Nodes:           make([]server.Node, 1),
		Permissioning: server.Perm{
			TlsCert:          cert,
			RegistrationCode: "",
			Address:          pAddr,
		},
	}

	// Register the node in a separate thread and notify when finished
	go func() {
		nodes, nodeIds, serverCert, gwCert := RegisterNode(def)
		def.Nodes = nodes
		def.TlsCert = []byte(serverCert)
		def.Gateway.TlsCert = []byte(gwCert)
		def.Topology = connect.NewCircuit(nodeIds)
		permDone <- struct{}{}
	}()

	// wait for gateway to connect
	<-gwConnected

	//poll server from gateway
	numPolls := 0
	for {
		if numPolls == 10 {
			t.Fatalf("Gateway could get cert from server")
		}
		numPolls++
		nodeHost, _ := gwComms.GetHost(nodeId.String())
		msg, err := gwComms.PollSignedCerts(nodeHost, &pb.Ping{})
		if err != nil {
			t.Errorf("Error on polling signed certs")
		}

		if msg.ServerCertPEM != "" && msg.GatewayCertPEM != "" {
			break
		}
	}

	//wait for server to finish
	<-permDone

	n := def.Nodes
	if len(n) < 1 {
		t.Errorf("Received empty network topology!")
	}
	if bytes.Compare(n[0].ID.Bytes(), nodeId.Bytes()) != 0 {
		t.Errorf("Received network topology with incorrect node ID!")
	}
	if n[0].Address != addr && strings.Replace(n[0].Address, "127.0.0.1",
		"0.0.0.0", -1) != addr {
		t.Errorf("Received network topology with incorrect node address!")
	}
	if n[0].TlsCert == nil {
		t.Errorf("Received network topology with incorrect node TLS cert!")
	}
}
