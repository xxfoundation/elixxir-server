///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package internal

import (
	"crypto/rand"
	"errors"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"
)

var dummyStates = [current.NUM_STATES]state.Change{
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
}

const MODP768 = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var instance *Instance

func TestMain(m *testing.M) {
	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}
	def := mockServerDef(m)
	sm := state.NewMachine(dummyStates)
	instance, _ = CreateServerInstance(def, impl, sm, "1.1.0")
	os.Exit(m.Run())
}

func TestRecoverInstance(t *testing.T) {
	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}
	def := mockServerDef(t)
	sm := state.NewMachine(dummyStates)

	msg := &mixmessages.RoundError{
		Id:     0,
		NodeId: id.NewIdFromUInt(uint64(0), id.Node, t).Marshal(),
		Error:  "test",
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Errorf("Failed to marshal test proto: %+v", err)
	}

	def.RecoveredErrorPath = "/tmp/test_err"

	err = utils.WriteFile(def.RecoveredErrorPath, b, utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Errorf("Failed to write to test file: %+v", err)
	}

	instance, _ = RecoverInstance(def, impl, sm, "1.1.0")
}

func TestInstance_GetResourceQueue(t *testing.T) {
	rq := initQueue()
	i := &Instance{resourceQueue: rq}

	if !reflect.DeepEqual(i.GetResourceQueue(), rq) {
		t.Errorf("Instance.GetResourceQueue: Returned incorrect " +
			"Resource Queue")
	}
}

func TestInstance_GetNetwork(t *testing.T) {
	n := &node.Comms{}
	i := &Instance{network: n}

	if !reflect.DeepEqual(i.GetNetwork(), n) {
		t.Errorf("Instance.GetResourceQueue: Returned incorrect " +
			"Network")
	}
}

func TestInstance_GetID(t *testing.T) {
	def := Definition{}
	def.ID = GenerateId(t)
	i := &Instance{definition: &def}

	if !reflect.DeepEqual(i.GetID(), def.ID) {
		t.Errorf("Instance.GetID: Returned incorrect " +
			"ID")
	}
}

func TestInstance_GetResourceMonitor(t *testing.T) {

	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}
	def := mockServerDef(t)
	m := state.NewMachine(dummyStates)
	tmpInstance, _ := CreateServerInstance(def, impl, m, "1.1.0")

	rm := tmpInstance.GetResourceMonitor()

	expectedMetric := measure.ResourceMetric{
		Time:          time.Unix(1, 2),
		MemAllocBytes: 1000,
		NumThreads:    10,
	}

	rm.Set(expectedMetric)

	if !tmpInstance.GetResourceMonitor().Get().Time.Equal(expectedMetric.Time) {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect time")
	}
	if tmpInstance.GetResourceMonitor().Get().NumThreads != expectedMetric.NumThreads {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect num threads")
	}
	if tmpInstance.GetResourceMonitor().Get().MemAllocBytes != expectedMetric.MemAllocBytes {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect mem allcoated")
	}

}

func mockServerDef(i interface{}) *Definition {
	nid := GenerateId(i)

	resourceMetric := measure.ResourceMetric{
		Time:          time.Now(),
		MemAllocBytes: 0,
		NumThreads:    0,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(resourceMetric)

	def := Definition{
		ID:              nid,
		ResourceMonitor: &resourceMonitor,
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           Flags{OverrideInternalIP: "0.0.0.0"},
		DevMode:         true,
	}

	def.Gateway.ID = nid.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)
	def.PrivateKey, _ = rsa.GenerateKey(rand.Reader, 1024)

	return &def
}

func TestCreateServerInstance(t *testing.T) {
	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}
	def := mockServerDef(t)
	m := state.NewMachine(dummyStates)
	_, err := CreateServerInstance(def, impl, m, "1.1.0")
	if err != nil {
		t.Logf("Failed to create a server instance")
		t.Fail()
	}
}

func createInstance(t *testing.T) (*Instance, *Definition) {
	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}
	def := mockServerDef(t)
	m := state.NewMachine(dummyStates)
	instance, err := CreateServerInstance(def, impl, m, "1.1.0")
	if err != nil {
		t.Logf("Failed to create a server instance")
		t.Fail()
	}
	return instance, def
}

func TestInstance_GetKeepBuffers(t *testing.T) {
	instance, def := createInstance(t)

	if def.Flags.KeepBuffers != instance.GetKeepBuffers() {
		t.Logf("Keep buffers is not expected Keep Buffers value")
		t.Fail()
	}
}

func TestInstance_GetMetricsLog(t *testing.T) {
	instance, def := createInstance(t)

	if def.MetricLogPath != instance.GetMetricsLog() {
		t.Logf("GetMetricLog returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetPrivKey(t *testing.T) {
	instance, def := createInstance(t)

	if def.PrivateKey != instance.GetPrivKey() {
		t.Logf("GetPrivKey returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetPubKey(t *testing.T) {
	instance, def := createInstance(t)

	if def.PublicKey != instance.GetPubKey() {
		t.Logf("GetPubKey returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetRegServerPubKey(t *testing.T) {
	instance, def := createInstance(t)

	if def.Permissioning.PublicKey != instance.GetRegServerPubKey() {
		t.Logf("GetMetricLog returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetRngStreamGen(t *testing.T) {
	instance, def := createInstance(t)

	if def.RngStreamGen != instance.GetRngStreamGen() {
		t.Logf("GetRngStreamGen returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetRoundManager(t *testing.T) {
	instance, _ := createInstance(t)

	if instance.roundManager != instance.GetRoundManager() {
		t.Logf("GetMetricLog returned unexpected value")
		t.Fail()
	}
}

func TestInstance_ReportCriticalError(t *testing.T) {
	instance, _ := createInstance(t)

	roundID := id.Round(987432)
	testErr := errors.New("Test error")
	instance.ReportRoundFailure(testErr, id.NewIdFromUInt(uint64(1), id.Node, t), roundID, false)
	//Test happy path

	//Test that if we send a different error it changes as expected

}

func TestInstance_GetDisableStreaming(t *testing.T) {
	instance, def := createInstance(t)

	if def.DisableStreaming != instance.GetDisableStreaming() {
		t.Logf("GetDisableStreaming() returned unexpected value")
		t.Fail()
	}
}

func panicHandler(g, m string, err error) {
	panic(g)
}

func TestInstance_OverridePhases(t *testing.T) {
	instance, _ := createInstance(t)
	gc := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 0)
	g := graphs.InitErrorGraph(gc)
	th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return errors.New("Failed intentionally")
	}
	p := phase.New(phase.Definition{
		Graph:               g,
		Type:                phase.Type(0),
		TransmissionHandler: th,
		Timeout:             3399921,
		DoVerification:      false,
	})
	overrides := map[int]phase.Phase{}
	overrides[0] = p
	instance.OverridePhases(overrides)
	if len(instance.phaseOverrides) != len(overrides) || instance.phaseOverrides[0].GetTimeout() != 3399921 {
		t.Error("failed to set overrides properly")
	}
}

func TestInstance_OverridePhasesAtRound(t *testing.T) {
	instance, _ := createInstance(t)
	gc := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 0)
	g := graphs.InitErrorGraph(gc)
	th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return errors.New("Failed intentionally")
	}
	p := phase.New(phase.Definition{
		Graph:               g,
		Type:                phase.Type(0),
		TransmissionHandler: th,
		Timeout:             3399921,
		DoVerification:      false,
	})
	overrides := map[int]phase.Phase{}
	overrides[0] = p
	instance.OverridePhasesAtRound(overrides, 3)
	if len(instance.phaseOverrides) != len(overrides) || instance.phaseOverrides[0].GetTimeout() != 3399921 {
		t.Error("failed to set overrides properly")
	}
	if instance.overrideRound != 3 {
		t.Errorf("Failed to set override round, expected: %d, got: %d", 3, instance.overrideRound)
	}
}

func TestInstance_GetOverrideRound(t *testing.T) {
	instance := Instance{
		overrideRound: 3,
	}
	if instance.GetOverrideRound() != 3 {
		t.Errorf("GetOverrideRound is broken; should have returned %d, instead got %d",
			3, instance.GetOverrideRound())
	}
}

func TestInstance_GetPhaseOverrides(t *testing.T) {
	gc := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 0)
	g := graphs.InitErrorGraph(gc)
	th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return errors.New("Failed intentionally")
	}
	p := phase.New(phase.Definition{
		Graph:               g,
		Type:                phase.Type(0),
		TransmissionHandler: th,
		Timeout:             83721,
		DoVerification:      false,
	})
	overrides := map[int]phase.Phase{}
	overrides[0] = p

	instance := Instance{
		phaseOverrides: overrides,
	}

	instanceOverrides := instance.GetPhaseOverrides()

	if len(instanceOverrides) != len(overrides) || instanceOverrides[0].GetTimeout() != 83721 {
		t.Error("Failed to get phase overrides set in instance")
	}
}
