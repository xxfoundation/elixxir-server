package io

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"strings"
	"testing"
	"time"
)

func GetMockServerInstance(t *testing.T) *server.Instance {
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

	nodeId = server.GenerateId(t)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	def := server.Definition{
		CmixGroup: grp,
		Nodes: []server.Node{
			{
				ID: nodeId,
			},
		},
		ID:              nodeId,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		PrivateKey:      serverRSAPriv,
		PublicKey:       serverRSAPub,
	}

	def.Permissioning.PublicKey = regPrivKey.GetPublic()
	nodeIDs := make([]*id.Node, 0)
	nodeIDs = append(nodeIDs, nodeId)
	def.Topology = connect.NewCircuit(nodeIDs)

	serverInstance, _ = server.CreateServerInstance(&def, NewImplementation, false)
	return serverInstance
}

func TestGetRoundBufferInfo_RoundsInBuffer(t *testing.T) {
	// This is actually an edge case: the number of available precomps is
	// greater than zero. This should only happen in production if the
	// communication between the gateway and the node breaks down.
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}

	// Not actually making a Round for concision
	c.Push(nil)
	availableRounds, err := GetRoundBufferInfo(c, time.Second)
	if err != nil {
		t.Error(err)
	}
	if availableRounds != 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}
}

func TestGetRoundBufferInfo_Timeout(t *testing.T) {
	// More than timeout case: length is zero and stays there
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	rbi, _ := GetRoundBufferInfo(c, 2*time.Millisecond)
	if rbi != 0 {
		t.Error("Round buffer info timeout should have resulted in an error")
	}
}

func TestGetRoundBufferInfo_LessThanTimeout(t *testing.T) {
	// Tests less than timeout case: length that's zero, then one,
	// should result in a length of one
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	before := time.Now()
	time.AfterFunc(200*time.Millisecond, func() {
		c.Push(nil)
	})
	availableRounds, err := GetRoundBufferInfo(c, time.Second)
	// elapsed time should be around 200 milliseconds,
	// because that's when the channel write happened
	after := time.Since(before)
	if after < 100*time.Millisecond || after > 400*time.Millisecond {
		t.Errorf("RoundBufferInfo result came in at an odd duration: %v", after)
	}
	if err != nil {
		t.Error(err)
	}
	if availableRounds != 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}
}

func TestGetCompletedBatch_Timeout(t *testing.T) {
	s := GetMockServerInstance(t)
	doneChan := make(chan struct{})

	var batch *mixmessages.Batch

	h, _ := connect.NewHost(s.GetID().NewGateway().String(), "test", nil, false, false)
	// Should timeout

	go func() {
		batch, _ = GetCompletedBatch(s, 40*time.Millisecond, &connect.Auth{
			IsAuthenticated: true,
			Sender:          h,
		})

		doneChan <- struct{}{}

	}()

	<-doneChan

	if len(batch.Slots) != 0 {
		t.Error("Should have gotten an error in the timeout case")
	}
}

func TestGetCompletedBatch_ShortWait(t *testing.T) {
	s := GetMockServerInstance(t)
	s.InitLastNode()

	// Should not timeout: writes to the completed rounds after an amount of
	// time
	var batch *mixmessages.Batch
	var err error

	doneChan := make(chan struct{})

	complete := &server.CompletedRound{
		RoundID:    42, //meaning of life
		Receiver:   make(chan services.Chunk),
		GetMessage: func(uint32) *mixmessages.Slot { return nil },
	}

	h, _ := connect.NewHost(s.GetID().NewGateway().String(), "test", nil, false, false)
	go func() {
		batch, err = GetCompletedBatch(s, 20*time.Millisecond, &connect.Auth{
			IsAuthenticated: true,
			Sender:          h,
		})
		doneChan <- struct{}{}
	}()

	time.After(5 * time.Millisecond)

	s.GetCompletedBatchQueue() <- complete

	complete.Receiver <- services.NewChunk(0, 3)

	close(complete.Receiver)

	<-doneChan

	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}

func TestGetCompletedBatch_BatchReady(t *testing.T) {
	s := GetMockServerInstance(t)
	s.InitLastNode()
	// If there's already a completed batch, the comm should get it immediately
	completedRoundQueue := s.GetCompletedBatchQueue()

	// Should not timeout: writes to the completed rounds after an amount of
	// time

	var batch *mixmessages.Batch
	var err error

	doneChan := make(chan struct{})

	complete := &server.CompletedRound{
		RoundID:    42, //meaning of life
		Receiver:   make(chan services.Chunk, 1),
		GetMessage: func(uint32) *mixmessages.Slot { return nil },
	}

	completedRoundQueue <- complete

	complete.Receiver <- services.NewChunk(0, 3)

	h, _ := connect.NewHost(s.GetID().NewGateway().String(), "test", nil, false, false)
	go func() {
		batch, err = GetCompletedBatch(s, 20*time.Millisecond, &connect.Auth{
			IsAuthenticated: true,
			Sender:          h,
		})
		doneChan <- struct{}{}
	}()

	close(complete.Receiver)

	<-doneChan

	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}

// Test that we receive authentication error when not authenticated
func TestGetCompletedBatch_NoAuth(t *testing.T) {
	s := GetMockServerInstance(t)
	h, _ := connect.NewHost(s.GetID().NewGateway().String(), "test", nil, false, false)
	_, err := GetCompletedBatch(s, 20*time.Millisecond, &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	})

	if err == nil {
		t.Errorf("Should have received authentication error")
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Errorf("Did not receive expected auth error.  Instead received: %+v", err)
	}
}

// Test that we receive authentication error when message is received from unexpected sender
func TestGetCompletedBatch_WrongSender(t *testing.T) {
	s := GetMockServerInstance(t)
	h, _ := connect.NewHost("test", "test", nil, false, false)
	_, err := GetCompletedBatch(s, 20*time.Millisecond, &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	})

	if err == nil {
		t.Errorf("Should have received authentication error")
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Errorf("Did not receive expected auth error.  Instead received: %+v", err)
	}
}
