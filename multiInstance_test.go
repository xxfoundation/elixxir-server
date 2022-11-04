////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package main

import (
	crand "crypto/rand"
	gorsa "crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"gitlab.com/elixxir/primitives/states"
	"math/big"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	nodeComms "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

var errWg = sync.WaitGroup{}

func Test_MultiInstance_N3_B8(t *testing.T) {
	elapsed := MultiInstanceTest(3, 32, makeMultiInstanceGroup(), false, false, t)

	t.Logf("Computational elapsed time for 3 Node, batch size 32, CPU multi-"+
		"instance test: %s", elapsed)
}

// fixme: find a way for this to work with precompTestBatch
//func Test_MultiInstance_PhaseErr(t *testing.T) {
//	elapsed := MultiInstanceTest(3, 32, makeMultiInstanceGroup(), false, true, t)
//
//	t.Logf("Computational elapsed time for 3 Node, batch size 32, error multi-"+
//		"instance test: %s", elapsed)
//}

func MultiInstanceTest(numNodes, batchSize int, grp *cyclic.Group, useGPU, errorPhase bool, t *testing.T) time.Duration {
	if errorPhase {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
	}

	jww.SetStdoutThreshold(jww.LevelDebug)

	if numNodes < 3 {
		t.Errorf("Multi Instance Test must have a minnimum of 3 nodes,"+
			" Received %v", numNodes)
	}

	// Get parameters
	portOffset := int(rand.Uint32() % 2000)
	viper.Set("useGPU", useGPU)

	defsLst := makeMultiInstanceParams(numNodes, 20000+portOffset, grp, useGPU, t)

	// Make user for sending messages
	userID := id.NewIdFromUInt(42, id.User, t)
	var baseKeys []*cyclic.Int
	for i := 0; i < numNodes; i++ {
		baseKey := grp.NewIntFromUInt(uint64(1000 + 5*i))
		baseKeys = append(baseKeys, baseKey)
	}

	// Build the instances
	var instances []*internal.Instance
	t.Logf("Building instances for %v nodes", numNodes)

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})

	for i := 0; i < numNodes; i++ {
		var instance *internal.Instance

		// Add handler for instance
		impl := func(i *internal.Instance) *nodeComms.Implementation {
			impl := io.NewImplementation(i)

			return impl
		}

		// Construct the state machine
		var testStates [current.NUM_STATES]state.Change
		// Create not started
		testStates[current.NOT_STARTED] = func(from current.Activity) error {
			curActivity, err := instance.GetStateMachine().WaitFor(1*time.Second,
				current.NOT_STARTED)
			if curActivity != current.NOT_STARTED || err != nil {
				t.Errorf("Server never transitioned to %v state: %+v",
					current.NOT_STARTED, err)
			}

			ok, err := instance.GetStateMachine().Update(current.WAITING)
			if !ok || err != nil {
				t.Errorf("Unable to transition to %v state: %+v",
					current.WAITING, err)
			}

			return nil
		}
		// Create waiting
		testStates[current.WAITING] = func(from current.Activity) error { return nil }
		// Create precomputing
		testStates[current.PRECOMPUTING] = func(from current.Activity) error {
			return node.Precomputing(instance)
		}
		// Create standby
		testStates[current.STANDBY] = func(from current.Activity) error { return nil }
		// Create realtime
		testStates[current.REALTIME] = func(from current.Activity) error {
			return node.Realtime(instance)
		}
		testStates[current.COMPLETED] = func(from current.Activity) error { return nil }
		testStates[current.ERROR] = func(from current.Activity) error {
			return node.Error(instance)
		}

		sm := state.NewMachine(testStates)

		instance, _ = internal.CreateServerInstance(defsLst[i], impl, sm, "1.1.0")
		err := instance.GetNetworkStatus().UpdateNodeConnections()
		if err != nil {
			t.Errorf("Failed to update node connections for node %d: %+v", i, err)
		}

		if errorPhase && i == 0 {
			gc := services.NewGraphGenerator(4,
				uint8(runtime.NumCPU()), 1, 0)
			g := graphs.InitErrorGraph(gc)
			th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
				return errors.New("Failed intentionally")
			}
			overrides := map[int]phase.Phase{}
			p := phase.New(phase.Definition{
				Graph:               g,
				Type:                phase.PrecompGeneration,
				TransmissionHandler: th,
				Timeout:             5 * time.Second,
				DoVerification:      false,
			})
			overrides[0] = p
			instance.OverridePhases(overrides)
			fmt.Println("I CAUSED A CRASH")
			errWg.Add(1)
			f := func(s string) {
				errWg.Done()
			}
			instance.OverridePanicWrapper(f, t)
		}

		instances = append(instances, instance)
	}

	t.Logf("Initilizing Network for %v nodes", numNodes)
	// Initialize the network for every instance
	for i, instance := range instances {
		instance.GetNetwork().DisableAuth()
		instance.Online = true
		params := connect.GetDefaultHostParams()
		params.AuthEnabled = false
		_, err := instance.GetNetwork().AddHost(&id.Permissioning,
			testUtil.NDF.Registration.Address, []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add permissioning host: %v", err)
		}
		instance.PopulateDummyUsers(true, grp)
		instance.AddDummyUserTesting(userID, baseKeys[i].Bytes(), grp, t)
	}

	t.Logf("Running the Queue for %v nodes", numNodes)
	// Begin every instance
	wg := sync.WaitGroup{}
	for _, instance := range instances {
		wg.Add(1)
		localInstance := instance
		go func() {
			time.Sleep(2 * time.Second)
			err := localInstance.Run()
			if err != nil {
				t.Errorf("uh-oh spaghetti-O's: %+v", err)
			}
			wg.Done()

		}()
	}
	wg.Wait()

	// Build topology
	ourTopology := make([][]byte, 0)
	for _, nodeInstance := range instances {
		ourTopology = append(ourTopology, nodeInstance.GetID().Marshal())
	}

	// Construct round info message
	roundInfoMsg := &mixmessages.RoundInfo{
		ID:                         1,
		UpdateID:                   1,
		State:                      uint32(current.PRECOMPUTING),
		BatchSize:                  uint32(batchSize),
		Topology:                   ourTopology,
		ResourceQueueTimeoutMillis: 5000,
		Timestamps:                 make([]uint64, states.NUM_STATES),
	}

	expectedBatch, ecrBatch, err := buildMockBatch(batchSize, grp, baseKeys, userID, roundInfoMsg)
	if err != nil {
		t.Errorf("%+v", err)
	}

	done := make(chan time.Time)
	go iterate(done, instances, t, ecrBatch, roundInfoMsg, errorPhase)
	start := <-done
	elapsed := time.Now().Sub(start)
	// Wait for last node to be ready to receive the batch
	completedBatch := &mixmessages.Batch{Slots: make([]*mixmessages.Slot, 0)}

	if !errorPhase {
		rid, err := instances[numNodes-1].GetCompletedBatchRID()
		if err != nil {
			t.Errorf("Unable to receive from CompletedBatchQueue: %+v", err)
		}

		cr, _ := instances[numNodes-1].GetCompletedBatch(rid)
		if cr != nil {
			completedBatch.Slots = cr.Round
		}
	}
	// --- BUILD PROBING TOOLS -------------------------------------------------
	// Get round buffers for probing
	var roundBuffs []*round.Buffer
	for _, instance := range instances {
		r, _ := instance.GetRoundManager().GetRound(1)
		roundBuffs = append(roundBuffs, r.GetBuffer())
	}

	// Build i/o map of permutations
	permutationMapping := make([]uint32, batchSize)
	for i := uint32(0); i < uint32(batchSize); i++ {
		slotIndex := i
		for _, buf := range roundBuffs {
			slotIndex = buf.Permutations[slotIndex]
		}
		permutationMapping[i] = slotIndex
	}

	// --- CHECK OUTPUTS -------------------------------------------------------
	found := 0
	for i := 0; i < batchSize; i++ {

		inputSlot := expectedBatch.Slots[i]
		outputSlot := completedBatch.Slots[permutationMapping[i]]
		success := true
		if grp.NewIntFromBytes(inputSlot.PayloadA).Cmp(grp.NewIntFromBytes(outputSlot.PayloadA)) != 0 {
			t.Errorf("Input slot %v permuted to slot %v payload A did "+
				"not match; \n Expected: %s \n Received: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.PayloadA).Text(16),
				grp.NewIntFromBytes(outputSlot.PayloadA).Text(16))
			success = false
		}

		if grp.NewIntFromBytes(inputSlot.PayloadB).Cmp(grp.NewIntFromBytes(outputSlot.PayloadB)) != 0 {
			t.Errorf("Input slot %v permuted to slot %v payload B did "+
				"not match; \n Expected: %s \n Received: %s", i, permutationMapping[i],
				grp.NewIntFromBytes(inputSlot.PayloadB).Text(16),
				grp.NewIntFromBytes(outputSlot.PayloadB).Text(16))
			success = false
		}

		if success {
			found++
		}
	}

	if found < batchSize {
		t.Fatalf("%v/%v of messages came out incorrect",
			batchSize-found, batchSize)
	} else {
		t.Logf("All messages received, passed")
	}

	// --- CHECK PRECOMPUTATION ------------------------------------------------

	// SHARE PHASE=
	pk := roundBuffs[0].CypherPublicKey.DeepCopy()
	// Test that all nodes have the same PK
	for itr, buf := range roundBuffs {
		pkNode := buf.CypherPublicKey.DeepCopy()
		if pkNode.Cmp(pk) != 0 {
			t.Errorf("Multinode instance test: node %v does not have "+
				"the same CypherPublicKey as node 1; node 1: %s, node %v: %s",
				itr+1, pk.Text(16), itr+1, pkNode.Text(16))
		}
	}

	// Test that the PK is the composition of the Zs
	for _, buf := range roundBuffs {
		Z := buf.Z.DeepCopy()
		pkOld := pk.DeepCopy()
		grp.RootCoprime(pkOld, Z, pk)
	}

	if pk.GetLargeInt().Cmp(grp.GetG()) != 0 {
		t.Errorf("Multinode instance test: inverse PK is not equal "+
			"to generator: Expected: %s, Received: %s",
			grp.GetG().Text(16), roundBuffs[0].CypherPublicKey.Text(16))
	}

	// Final result
	// Traverse the nodes to find the final precomputation for each slot

	// Create precomp buffer
	payloadAPrecomps := make([]*cyclic.Int, batchSize)
	payloadBPrecomps := make([]*cyclic.Int, batchSize)

	for i := 0; i < batchSize; i++ {
		payloadAPrecomps[i] = grp.NewInt(1)
		payloadBPrecomps[i] = grp.NewInt(1)
	}

	// Precomp Decrypt
	for i := uint32(0); i < uint32(batchSize); i++ {
		for _, buf := range roundBuffs {
			grp.Mul(payloadAPrecomps[i], buf.R.Get(i), payloadAPrecomps[i])
			grp.Mul(payloadBPrecomps[i], buf.U.Get(i), payloadBPrecomps[i])
		}
	}

	// Precomp permute
	for i := uint32(0); i < uint32(batchSize); i++ {
		slotIndex := i
		for _, buf := range roundBuffs {
			grp.Mul(payloadAPrecomps[i], buf.S.Get(slotIndex), payloadAPrecomps[i])
			grp.Mul(payloadBPrecomps[i], buf.V.Get(slotIndex), payloadBPrecomps[i])
			slotIndex = buf.Permutations[slotIndex]
		}
		grp.Inverse(payloadAPrecomps[i], payloadAPrecomps[i])
		grp.Inverse(payloadBPrecomps[i], payloadBPrecomps[i])
	}

	for i := 0; i < batchSize; i++ {
		resultPayloadA := roundBuffs[len(roundBuffs)-1].PayloadAPrecomputation.Get(permutationMapping[i])
		if payloadAPrecomps[i].Cmp(resultPayloadA) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadA slot %v "+
				"incorrect; Expected: %s, Received: %s", i,
				payloadAPrecomps[i].Text(16), resultPayloadA.Text(16))
		}
		resultPayloadB := roundBuffs[len(roundBuffs)-1].PayloadBPrecomputation.Get(permutationMapping[i])
		if payloadBPrecomps[i].Cmp(resultPayloadB) != 0 {
			t.Errorf("Multinode instance test: precomputation for payloadB slot %v "+
				"incorrect; Expected: %s, Received: %s", i,
				payloadBPrecomps[i].Text(16), resultPayloadB.Text(16))
		}
	}

	return elapsed
}

// buildMockBatch
func buildMockBatch(batchSize int, grp *cyclic.Group, baseKeys []*cyclic.Int,
	userID *id.ID, ri *mixmessages.RoundInfo) (*pb.Batch, *pb.Batch, error) {
	// Build a batch to send to first node
	expectedBatch := &mixmessages.Batch{}
	ecrBatch := &mixmessages.Batch{}

	kmacHash, err2 := hash.NewCMixHash()
	if err2 != nil {
		return &pb.Batch{}, &pb.Batch{}, errors.Errorf("Could not get KMAC hash: %+v", err2)
	}
	for i := 0; i < batchSize; i++ {
		// Make the salt
		salt := make([]byte, 32)
		binary.BigEndian.PutUint64(salt[0:8], uint64(100+6*i))

		// Make the payload
		primeLength := uint64(grp.GetP().ByteLen())
		payloadA := grp.NewIntFromUInt(uint64(1 + i)).LeftpadBytes(primeLength)
		payloadB := grp.NewIntFromUInt(uint64((513 + i) * 256)).LeftpadBytes(primeLength)

		// Make the message

		msg := format.NewMessage(int(primeLength))
		msg.SetPayloadA(payloadA)
		msg.SetPayloadB(payloadB)

		// Encrypt the message
		ecrMsg := cmix.ClientEncrypt(grp, msg, salt, baseKeys, id.Round(ri.ID))
		kmacs := cmix.GenerateKMACs(salt, baseKeys, id.Round(ri.ID), kmacHash)

		// Make the slot
		ecrSlot := &mixmessages.Slot{}
		ecrSlot.PayloadA = ecrMsg.GetPayloadA()
		ecrSlot.PayloadB = ecrMsg.GetPayloadB()
		ecrSlot.SenderID = userID.Bytes()
		ecrSlot.Salt = salt
		ecrSlot.KMACs = kmacs

		ecrBatch.Slots = append(ecrBatch.Slots, ecrSlot)
		ecrBatch.Round = ri

		slot := &mixmessages.Slot{}
		slot.PayloadA = msg.GetPayloadA()
		slot.PayloadB = msg.GetPayloadB()
		slot.SenderID = userID.Bytes()
		slot.Salt = salt
		expectedBatch.Slots = append(expectedBatch.Slots, slot)
	}

	return expectedBatch, ecrBatch, nil
}

func iterate(done chan time.Time, nodes []*internal.Instance, t *testing.T,
	ecrBatch *pb.Batch, roundInfoMsg *mixmessages.RoundInfo, errorPhase bool) {
	time.Sleep(2 * time.Second)
	// Define a mechanism to wait until the next state
	asyncWaitUntil := func(wg *sync.WaitGroup, until current.Activity, node *internal.Instance) {
		wg.Add(1)
		go func() {
			success, err := node.GetStateMachine().WaitForUnsafe(until, 5*time.Second, t)
			if !success {
				jww.FATAL.Printf("Wait for node %s to enter state %s failed: %s", node.GetID(), until.String(), err)
			} else {
				wg.Done()
			}
		}()

	}

	// Wait until all nodes are started
	wg := sync.WaitGroup{}

	// Parse through the nodes prepping them for rounds
	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.WAITING, nodeInstance)
	}

	wg.Wait()
	// Mocking permissioning server signing message
	signRoundInfo(roundInfoMsg)

	// Get starting time for benchmark
	start := time.Now()

	for index, nodeInstance := range nodes {
		_, err := nodeInstance.GetNetworkStatus().RoundUpdate(roundInfoMsg)
		if err != nil {
			t.Errorf("Failed to updated network instance for new round info: %v", err)
		}

		// Send the round info to the instance (to be handled internally later)
		err = nodeInstance.GetCreateRoundQueue().Send(roundInfoMsg)
		if err != nil {
			t.Errorf("Unable to send to RealtimeRoundQueue for node %d: %+v", index, err)
		}

		// Begin the PRECOMPUTING state
		ok, err := nodeInstance.GetStateMachine().Update(current.PRECOMPUTING)
		if !ok || err != nil {
			t.Errorf("Cannot move to precomputing state: %+v", err)
		}

	}

	if errorPhase {
		fmt.Println("GGGGGGG")
		errWg.Wait()
		fmt.Println("BBBBBB")
		for _, nodeInstance := range nodes {
			err := nodeInstance.GetResourceQueue().Kill(1 * time.Second)
			if err != nil {
				t.Errorf("Node failed to kill: %s", err)
			}
		}

		done <- start
		return
	}

	// Read to look in permissioning, manually do steps
	// Parse through the nodes prepping them for rounds
	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.STANDBY, nodeInstance)
	}

	wg.Wait()
	for i := len(nodes) - 1; i >= 0; i-- {
		nodeInstance := nodes[i]
		// Send info to the realtime round queue
		err := nodeInstance.GetRealtimeRoundQueue().Send(roundInfoMsg)
		if err != nil {
			t.Errorf("Unable to send to RealtimeRoundQueue: %+v", err)
		}
		ok, err := nodeInstance.GetStateMachine().Update(current.REALTIME)
		if !ok || err != nil {
			t.Errorf("Failed to update to realtime: %+v", err)
			fmt.Printf("Failed to update to realtime: %+v\n", err)
		}
	}

	err := io.HandleRealtimeBatch(nodes[0], ecrBatch, io.PostPhase)
	if err != nil {
		t.Errorf("Unable to handle realtime batch: %+v", err)
	}

	for _, nodeInstance := range nodes {
		asyncWaitUntil(&wg, current.COMPLETED, nodeInstance)
	}

	wg.Wait()
	done <- start
}

// generateCert eturns a self-signed cert and key for dummy tls comms,
// this is mostly cribbed from:
//
//	https://golang.org/src/crypto/tls/generate_cert.go
func generateCert() ([]byte, []byte) {
	priv, err := gorsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		jww.FATAL.Panicf("Failed to generate private key: %v", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(10 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := crand.Int(crand.Reader, serialNumberLimit)
	if err != nil {
		jww.FATAL.Panicf("Failed to generate serial number: %v", err)
	}

	usage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	usage |= x509.KeyUsageCertSign
	extUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Dummy Key"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              usage,
		ExtKeyUsage:           extUsage,
		BasicConstraintsValid: true,
	}

	template.IPAddresses = append(template.IPAddresses,
		net.ParseIP("127.0.0.1"))
	//template.DNSNames = append(template.DNSNames, "localhost")

	template.IsCA = true

	derBytes, err := x509.CreateCertificate(crand.Reader, &template,
		&template, &priv.PublicKey, priv)
	if err != nil {
		jww.FATAL.Panicf("Failed to create certificate: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	crtOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE",
		Bytes: derBytes})
	privOut := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY",
		Bytes: privBytes})
	return crtOut, privOut
}

func makeMultiInstanceParams(numNodes, portStart int, grp *cyclic.Group, useGPU bool, t *testing.T) []*internal.Definition {

	// Generate IDs and addresses
	var nidLst []*id.ID
	var nodeLst []internal.Node
	addrFmt := "localhost:%03d"
	for i := 0; i < numNodes; i++ {
		// Generate id
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nidLst = append(nidLst, nodeID)
		// Generate address
		addr := fmt.Sprintf(addrFmt, i+portStart)

		n := internal.Node{
			ID:      nodeID,
			Address: addr,
		}
		nodeLst = append(nodeLst, n)

	}

	networkDef := buildNdf(nodeLst, grp)

	// Generate parameters list
	var defLst []*internal.Definition

	cert, privKey := generateCert()

	for i := 0; i < numNodes; i++ {
		gatewayID := nidLst[i].DeepCopy()
		gatewayID.SetType(id.Gateway)

		def := internal.Definition{
			ID: nidLst[i],
			Flags: internal.Flags{
				KeepBuffers: true,
				UseGPU:      useGPU,
			},
			TlsCert: cert,
			Gateway: internal.GW{
				ID:      gatewayID,
				TlsCert: nil,
				Address: "",
			},
			ResourceMonitor:    &measure.ResourceMonitor{},
			FullNDF:            networkDef,
			PartialNDF:         networkDef,
			ListeningAddress:   nodeLst[i].Address,
			MetricsHandler:     func(i *internal.Instance, roundID id.Round) error { return nil },
			RecoveredErrorPath: fmt.Sprintf("/tmp/err_%d", i),
			GraphGenerator:     services.NewGraphGenerator(4, 1, 4, 1.0),
			RngStreamGen: fastRNG.NewStreamGenerator(10000,
				uint(runtime.NumCPU()), csprng.NewSystemRNG),
			DevMode: true,
		}

		cryptoPrivRSAKey, _ := tls.LoadRSAPrivateKey(string(privKey))

		def.PrivateKey = &rsa.PrivateKey{*cryptoPrivRSAKey}

		defLst = append(defLst, &def)
	}

	return defLst
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

func makeMultiInstanceGroup4k() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6BF12FFA06D98A0864D87602733EC86A64521F2B18177B200CBBE117577A615D6C770988C0BAD946E208E24FA074E5AB3143DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D788719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA993B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199FFFFFFFFFFFFFFFF"

	return cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))
}

// buildNdf builds the ndf used for definitions
func buildNdf(nodeLst []internal.Node, grp *cyclic.Group) *ndf.NetworkDefinition {
	// Pull the node id's out of nodeList
	ndfNodes := make([]ndf.Node, 0)
	for _, ourNode := range nodeLst {
		tmpNode := ndf.Node{
			ID:             ourNode.ID.Bytes(),
			Address:        ourNode.Address,
			TlsCertificate: string(ourNode.TlsCert),
		}
		ndfNodes = append(ndfNodes, tmpNode)

	}

	// Build a group
	group := ndf.Group{
		Prime:      grp.GetP().TextVerbose(16, 0),
		SmallPrime: "2",
		Generator:  grp.GetG().TextVerbose(16, 0),
	}

	// Construct an ndf
	return &ndf.NetworkDefinition{
		Timestamp: time.Time{},
		Nodes:     ndfNodes,
		E2E:       group,
		CMIX:      group,
	}

}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("Couldn't load private key: %+v", err)
	}

	ourPrivateKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.SignRsa(ri, ourPrivateKey)
	return nil
}
