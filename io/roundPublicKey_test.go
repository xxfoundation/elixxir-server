////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// FIXME: this import list makes it feel like the api is spaghetti
import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"os"
	"testing"
	"time"
)

var nodeAddrList *services.NodeAddressList

const primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var instances []*server.Instance
var grp *cyclic.Group

func TestMain(m *testing.M) {
	// We need 3 servers, prev, cur, next
	addrFmt := "localhost:600%d"
	cnt := 3
	servers := make([]node.ServerHandler, cnt)
	// FIXME: What justifies this NodeAddressList design?
	// This API is painful to work with. Should probably go in comms...
	addrs := make([]services.NodeAddress, cnt)
	// FIXME: we shouldn't need to do this for a comms test
	grp = cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(1283))
	instances = make([]*server.Instance, cnt)
	for i := 0; i < cnt; i++ {
		addrs[i] = services.NodeAddress{
			Address: fmt.Sprintf(addrFmt, i),
			Cert:    "",
			Id:      id.Node{},
		}
		// This also seems like overkill for a comms test
		instances[i] = server.CreateServerInstance(grp,
			&globals.UserMap{})
		servers[i] = NewImplementation(instances[i])
		go node.StartServer(addrs[i].Address, servers[i], "", "")
	}
	nodeAddrList = services.NewNodeAddressList(addrs, 1)
	os.Exit(m.Run())
}

func TestPostRoundPublicKey(t *testing.T) {

	rm := instances[2].GetRoundManager()
	roundID := id.Round(42)

	round := round.New(grp, roundID, phases, nil, 0, 1)
	rm.AddRound(round)

	roundPubKey := grp.NewIntFromUInt(42)

	err := TransmitRoundPublicKey(roundPubKey, 42,
		nodeAddrList)

	// TODO: Cycle through all the servers and ensure the
	// roundPublicKey is set to the same value.
	if err != nil {
		t.Errorf("%v", err)
	}
}
