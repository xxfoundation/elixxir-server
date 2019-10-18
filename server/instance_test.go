package server

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
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
	def := mockServerDef(grp)
	instance = CreateServerInstance(def)
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
	n := &node.NodeComms{}
	i := &Instance{network: n}

	if !reflect.DeepEqual(i.GetNetwork(), n) {
		t.Errorf("Instance.GetResourceQueue: Returned incorrect " +
			"Network")
	}
}

func TestInstance_GetID(t *testing.T) {
	def := Definition{}
	def.ID = GenerateId(true)
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
	def.Topology = circuit.New(nodeIDs)
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

	def := mockServerDef(grp)
	i := CreateServerInstance(def)

	rm := i.GetResourceMonitor()

	expectedMetric := measure.ResourceMetric{
		Time:          time.Unix(1, 2),
		MemAllocBytes: 1000,
		NumThreads:    10,
	}

	rm.Set(&expectedMetric)

	if !i.GetResourceMonitor().Get().Time.Equal(expectedMetric.Time) {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect time")
	}
	if i.GetResourceMonitor().Get().NumThreads != expectedMetric.NumThreads {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect num threads")
	}
	if i.GetResourceMonitor().Get().MemAllocBytes != expectedMetric.MemAllocBytes {
		t.Errorf("Instance.GetResourceMonitor: Returned incorrect mem allcoated")
	}

}

func mockServerDef(grp *cyclic.Group) *Definition {
	nid := GenerateId(true)

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
