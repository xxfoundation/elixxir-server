package realtime

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
)

// Module that implements Keygen, along with helper methods
type KeygenSubStream struct {
	// User IDs for each position in the batch
	grp *cyclic.Group
	// Inputs: user IDs and salts (required for key generation)
	users []*id.User
	salts [][]byte

	// Output: keys
	keys *cyclic.IntBuffer
}

// LinkStream This Link doesn't conform to the Stream interface because KeygenSubStream
// isn't meant for use alone in a graph
// For salts and users: the slices don't have to point to valid underlying data
// at Link time, but they should represent an area that'll be filled with valid
// data or space for data when the cryptop runs
func (k *KeygenSubStream) LinkStream(grp *cyclic.Group,
	inSalts [][]byte, inUsers []*id.User, outKeys *cyclic.IntBuffer) {
	k.grp = grp
	k.salts = inSalts
	k.users = inUsers
	k.keys = outKeys
}

//Returns the substream, used to return an embedded struct off an interface
func (k *KeygenSubStream) getSubStream() *KeygenSubStream {
	return k
}

// *KeygenSubStream conforms to this interface, so pass the embedded substream
// struct to this module when you're using it
type keygenSubStreamInterface interface {
	getSubStream() *KeygenSubStream
}

var Keygen = services.Module{
	Adapt: func(s services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		streamInterface, ok := s.(keygenSubStreamInterface)
		keygen, ok2 := cryptop.(cryptops.KeygenPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		kss := streamInterface.getSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			user, err := globals.Users.GetUser(kss.users[i])
			if err != nil {
				return err
			}
			keygen(kss.grp, kss.salts[i], user.BaseKey, kss.keys.Get(i))
		}

		return nil
	},
	Cryptop:        cryptops.Keygen,
	InputSize:      services.AUTO_INPUTSIZE,
	StartThreshold: 0,
	Name:           "Keygen",
	NumThreads:     8,
}
