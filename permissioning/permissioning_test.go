////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/services"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var nodeId *id.Node
var permComms *registration.RegistrationComms
var gwComms *gateway.GatewayComms

// Dummy implementation of permissioning server --------------------------------
type mockPermission struct{}

func (i *mockPermission) RegisterUser(registrationCode string, Y, P, Q,
	G []byte) (hash, R, S []byte, err error) {
	return nil, nil, nil, nil
}
func (i *mockPermission) RegisterNode(ID []byte,
	NodeTLSCert, GatewayTLSCert, RegistrationCode, Addr string) error {

	go func() {
		err := permComms.ConnectToNode(nodeId, Addr, nil)
		if err != nil {
			panic(err)
		}
		nodeTop := make([]*pb.NodeInfo, 0)
		nodeTop = append(nodeTop, &pb.NodeInfo{
			Id:             nodeId.Bytes(),
			Index:          0,
			IpAddress:      Addr,
			ServerTlsCert:  "a",
			GatewayTlsCert: "b",
		})
		nwTop := &pb.NodeTopology{
			Topology: nodeTop,
		}
		err = permComms.SendNodeTopology(nodeId, nwTop)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

// Dummy implementation of gateway server --------------------------------------
type mockGateway struct{}

func (*mockGateway) CheckMessages(userID *id.User, messageID string) ([]string, bool) {
	return nil, false
}

func (*mockGateway) GetMessage(userID *id.User, msgID string) (*pb.Slot, bool) {
	return nil, false
}

func (*mockGateway) PutMessage(message *pb.Slot) bool {
	return false
}

func (*mockGateway) RequestNonce(message *pb.NonceRequest) (*pb.Nonce, error) {
	return nil, nil
}

func (*mockGateway) ConfirmNonce(message *pb.DSASignature) (*pb.
	RegistrationConfirmation, error) {
	return nil, nil
}

// -----------------------------------------------------------------------------

// Full-stack happy path test for the node registration logic
func TestRegisterNode(t *testing.T) {

	gwConnected := make(chan struct{})
	permDone := make(chan struct{})

	nodeId = id.NewNodeFromUInt(uint64(0), t)
	addr := fmt.Sprintf("0.0.0.0:%d", 6000+rand.Intn(1000))

	// Initialize permissioning server
	pAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	pHandler := registration.Handler(&mockPermission{})
	permComms = registration.StartRegistrationServer(pAddr, pHandler, nil, nil)

	gAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))
	gHandler := gateway.Handler(&mockGateway{})
	gwComms = gateway.StartGateway(gAddr, gHandler, nil, nil)

	go func() {
		time.Sleep(1 * time.Second)
		err := gwComms.ConnectToNode(nodeId, addr, nil)
		if err != nil {
			t.Fatalf("Gateway could not connect to node")
		}
		gwConnected <- struct{}{}
	}()

	// Initialize definition
	def := &server.Definition{
		Flags:         server.Flags{},
		ID:            nodeId,
		DsaPublicKey:  nil,
		DsaPrivateKey: nil,
		TlsCert:       nil,
		TlsKey:        nil,
		Address:       addr,
		LogPath:       "",
		MetricLogPath: "",
		Gateway: server.GW{
			Address: gAddr,
			TlsCert: nil,
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
			TlsCert:          nil,
			RegistrationCode: "",
			Address:          pAddr,
		},
	}

	// Register the node in a separate thread and notify when finished
	go func() {
		nodes, serverCert, gwCert := RegisterNode(def)
		def.Nodes = nodes
		def.TlsCert = []byte(serverCert)
		def.Gateway.TlsCert = []byte(gwCert)
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
		msg, err := gwComms.PollSignedCerts(nodeId, &pb.Ping{})
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