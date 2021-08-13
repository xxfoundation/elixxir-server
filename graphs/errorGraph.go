///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"github.com/pkg/errors"
	"git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/crypto/cryptops"
	"git.xx.network/elixxir/crypto/cyclic"
	"git.xx.network/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Decrypt phase
// Decrypt phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// DecryptStream holds data containing keys and inputs used by decrypt
type ErrorStream struct {
}

// GetName returns stream name
func (ds *ErrorStream) GetName() string {
	return "ErrorStream"
}

// Link binds stream to state objects in round
func (ds *ErrorStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {}

// Input initializes stream inputs from slot
func (ds *ErrorStream) Input(index uint32, slot *mixmessages.Slot) error {
	return nil
}

// Output returns a cmix slot message
func (ds *ErrorStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

// DecryptElgamal is the sole module in Precomputation Decrypt implementing cryptops.Elgamal
var ErrorModule = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		return errors.New("Intentionally errored ErrorStream")
	},
	Cryptop:    cryptops.ElGamal,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "ErrorStream",
}

// InitDecryptGraph is called to initialize the graph. Conforms to graphs.Initialize function type
func InitErrorGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("ErrorGraph", &ErrorStream{})

	errorModule := ErrorModule.DeepCopy()

	g.First(errorModule)
	g.Last(errorModule)

	return g
}
