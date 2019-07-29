package permissioning

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
)

// Stringer object for Permissioning connection ID
type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// Perform the Node registration process with the Permissioning Server
func RegisterNode(def *server.Definition) {

	// Channel for signaling completion of Node registration
	ch := make(chan *pb.NodeTopology)

	// Assemble the Comms callback interface
	impl := node.NewImplementation()
	impl.Functions.DownloadTopology = func(topology *pb.NodeTopology) {
		// Signal completion of Node registration
		ch <- topology
	}

	// Start Node communication server
	network := node.StartNode(def.Address, impl, def.TlsCert, def.TlsKey)
	permissioningId := ConnAddr("permissioning")

	// Connect to the Permissioning Server
	err := network.ConnectToRegistration(permissioningId,
		def.Permissioning.Address, def.Permissioning.TlsCert)
	if err != nil {
		jww.FATAL.Panicf("Unable to initiate Node registration: %+v",
			errors.New(err.Error()))
	}

	// Attempt Node registration
	err = network.SendNodeRegistration(permissioningId,
		&pb.NodeRegistration{
			ID:               def.ID.Bytes(),
			NodeTLSCert:      string(def.TlsCert),
			GatewayTLSCert:   string(def.Gateway.TlsCert),
			RegistrationCode: def.Permissioning.RegistrationCode,
		})
	if err != nil {
		jww.FATAL.Panicf("Unable to send Node registration: %+v",
			errors.New(err.Error()))
	}

	// Wait for Node registration to complete
	select {
	case topology := <-ch:
		// Shut down the Comms server
		network.Shutdown()

		// Integrate the topology with the Definition
		def.Nodes = make([]server.Node, len(topology.Topology))
		for _, n := range topology.Topology {

			// Build Node information
			def.Nodes[n.Index] = server.Node{
				ID:      id.NewNodeFromBytes(n.Id),
				TlsCert: []byte(n.TlsCert),
				Address: n.IpAddress,
			}

			// Update Cert for this Node
			if bytes.Compare(n.Id, def.ID.Bytes()) == 0 {
				def.TlsCert = []byte(n.TlsCert)
			}
		}
	}
}
