package server

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"os"
	"reflect"
	"testing"
	"time"
)

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
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2))
	def := mockServerDef(m, grp)
	instance, _ = CreateServerInstance(def, NewImplementation, false)
	os.Exit(m.Run())
}

func TestInstance_GetGroup(t *testing.T) {
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2))
	if instance.GetGroup().GetFingerprint() != grp.GetFingerprint() {
		t.Errorf("Instance.GetGroup: Returned incorrect group")
	}
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

func TestInstance_Topology(t *testing.T) {
	var nodeIDs []*id.Node

	//Build IDs
	for i := 0; i < 3; i++ {
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	def := Definition{}
	def.Topology = connect.NewCircuit(nodeIDs)
	def.ID = nodeIDs[2]
	i := &Instance{definition: &def}

	if !reflect.DeepEqual(i.GetTopology(), def.Topology) {
		t.Errorf("Instance.GetTopology: Returned incorrect " +
			"Topology")
	}

	if i.IsFirstNode() {
		t.Errorf("I am not first node!")
	}
	if !i.IsLastNode() {
		t.Errorf("I should be last node!")
	}
}

func TestInstance_GetResourceMonitor(t *testing.T) {

	def := mockServerDef(t, grp)
	tmpInstance, _ := CreateServerInstance(def, NewImplementation, false)

	rm := tmpInstance.GetResourceMonitor()

	expectedMetric := measure.ResourceMetric{
		Time:          time.Unix(1, 2),
		MemAllocBytes: 1000,
		NumThreads:    10,
	}

	rm.Set(&expectedMetric)

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

func mockServerDef(i interface{}, grp *cyclic.Group) *Definition {
	nid := GenerateId(i)

	resourceMetric := measure.ResourceMetric{
		Time:          time.Now(),
		MemAllocBytes: 0,
		NumThreads:    0,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&resourceMetric)

	def := Definition{
		ID:              nid,
		CmixGroup:       grp,
		ResourceMonitor: &resourceMonitor,
	}

	return &def
}

func TestCreateServerInstance(t *testing.T) {
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2))
	def := mockServerDef(t, grp)
	_, err := CreateServerInstance(def, NewImplementation)
	if err != nil {
		t.Logf("Failed to create a server instance")
		t.Fail()
	}
}

func createInstance(t *testing.T) (*Instance, *Definition) {
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2))
	def := mockServerDef(t, grp)
	instance, err := CreateServerInstance(def, NewImplementation)
	if err != nil {
		t.Logf("Failed to create a server instance")
		t.Fail()
	}
	return instance, def
}

func TestInstance_GetBatchSize(t *testing.T) {
	instance, def := createInstance(t)

	if def.BatchSize != instance.GetBatchSize() {
		t.Logf("BatchSize is not = to batchsize from Def")
		t.Fail()
	}
}

// Need to modify mockserver to add this ip into it, or else it will panic since we have nothing to return
//func TestInstance_GetIP(t *testing.T) {
//	instance, _ := createInstance(t)
//	//addrWithPort := def.Nodes[instance.GetTopology().GetNodeLocation(instance.GetID())].Address
//	//instance.GetIP()
//	//expectedIP := strings.Split(addrWithPort, ":")[0]
//	//if expectedIP != instance.GetIP() {
//	//	t.Logf("IP is not expected IP value")
//	//	t.Fail()
//	//}
//}

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

func TestInstance_GetTopology(t *testing.T) {
	instance, def := createInstance(t)

	if def.Topology != instance.GetTopology() {
		t.Logf("GetTopology returned unexpected value")
		t.Fail()
	}
}

func TestInstance_GetUserRegistry(t *testing.T) {
	instance, def := createInstance(t)

	if def.UserRegistry != instance.GetUserRegistry() {
		t.Logf("GetTopology returned unexpected value")
		t.Fail()
	}
}

// We need to modify the mockServerDef for this to work?
//func TestInstance_IsFirstNode(t *testing.T) {
//	instance, def := createInstance(t)
//
//	if def.Topology.IsFirstNode(def.ID) != instance.IsFirstNode() {
//		t.Logf("IsFirstNode returned unexpected value")
//		t.Fail()
//	}
//}
//
//func TestInstance_IsLastNode(t *testing.T) {
//	instance, def := createInstance(t)
//
//	if def.Topology.IsLastNode(def.ID) != instance.IsLastNode() {
//		t.Logf("IsLastNode returned unexpected value")
//		t.Fail()
//	}
//}

func TestInstance_IsRegistrationAuthenticated(t *testing.T) {
	instance, def := createInstance(t)

	if def.Flags.SkipReg != instance.IsRegistrationAuthenticated() {
		t.Logf("IsRegistrationAuthenticated() returned unexpected value")
		t.Fail()
	}
}
