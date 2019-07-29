package server

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/conf"
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
	grp := cyclic.NewGroup(prime, large.NewInt(2), large.NewInt(1283))
	instance = mockServerInstance(grp)
	os.Exit(m.Run())
}

func TestInstance_GetGroup(t *testing.T) {
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2), large.NewInt(1283))
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
	nid := GenerateId()
	params := conf.Params{}
	i := &Instance{params: &params,
		thisNode: nid}

	if !reflect.DeepEqual(i.GetID(), nid) {
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
	top := circuit.New(nodeIDs)
	i := &Instance{topology: top, thisNode: nodeIDs[2]}

	if !reflect.DeepEqual(i.GetTopology(), top) {
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

func TestInstance_BadNodeID(t *testing.T) {
	prime := large.NewIntFromString(MODP768, 16)
	grp := cyclic.NewGroup(prime, large.NewInt(2), large.NewInt(1283))
	primeString := grp.GetP().TextVerbose(16, 0)

	smallprime := grp.GetQ().TextVerbose(16, 0)
	generator := grp.GetG().TextVerbose(16, 0)

	cmix := map[string]string{
		"prime":      primeString,
		"smallprime": smallprime,
		"generator":  generator,
	}

	params := conf.Params{
		Node: conf.Node{
			Ids: []string{"TWFuIGlzIGRpc3Rpbmd1aXNoZWQsIG5vdCBv***="},
		},
		Groups: conf.Groups{
			CMix: cmix,
		},
	}

	defer func() {
		if rec := recover(); rec == nil {
			t.Logf("Caught expected panic on bad node id")
		}
	}()

	instance := CreateServerInstance(&params, &globals.UserMap{}, nil, nil, &measure.ResourceMonitor{})

	if instance != nil {
		t.Errorf("BadeNode ID, so Instance should not have returned!!")
	}
}

func TestInstance_GetResourceMonitor(t *testing.T) {
	nid := GenerateId()
	params := conf.Params{}
	i := &Instance{params: &params,
		thisNode: nid, lastResourceMonitor: &measure.ResourceMonitor{}}

	rm := i.GetLastResourceMonitor()

	expectedMetric := measure.ResourceMetric{
		Time:              time.Unix(1, 2),
		MemoryAllocated:   "1000",
		NumThreads:        10,
		HighestMemThreads: "abc",
	}

	rm.Set(&expectedMetric)

	if !i.GetLastResourceMonitor().Get().Time.Equal(expectedMetric.Time) {
		t.Errorf("Instance.GetLastResourceMonitor: Returned incorrect time")
	}
	if i.GetLastResourceMonitor().Get().HighestMemThreads != expectedMetric.HighestMemThreads {
		t.Errorf("Instance.GetLastResourceMonitor: Returned incorrect mem threads")
	}
	if i.GetLastResourceMonitor().Get().NumThreads != expectedMetric.NumThreads {
		t.Errorf("Instance.GetLastResourceMonitor: Returned incorrect num threads")
	}
	if i.GetLastResourceMonitor().Get().MemoryAllocated != expectedMetric.MemoryAllocated {
		t.Errorf("Instance.GetLastResourceMonitor: Returned incorrect mem allcoated")
	}

}

func mockServerInstance(grp *cyclic.Group) *Instance {
	primeString := grp.GetP().TextVerbose(16, 0)

	smallprime := grp.GetQ().TextVerbose(16, 0)
	generator := grp.GetG().TextVerbose(16, 0)

	nid := GenerateId()

	cmix := map[string]string{
		"prime":      primeString,
		"smallprime": smallprime,
		"generator":  generator,
	}

	params := conf.Params{
		Node: conf.Node{
			Ids: []string{nid.String()},
		},
		Groups: conf.Groups{
			CMix: cmix,
		},
	}
	resourceMetric := measure.ResourceMetric{
		Time:              time.Now(),
		MemoryAllocated:   "",
		NumThreads:        0,
		HighestMemThreads: "",
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&resourceMetric)
	instance := CreateServerInstance(&params, &globals.UserMap{}, nil, nil, &measure.ResourceMonitor{})

	return instance
}
