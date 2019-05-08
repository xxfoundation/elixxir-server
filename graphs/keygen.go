package graphs

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
)

// Module that implements Keygen, along with helper methods
type KeygenSubStream struct {
	// Server state that's needed for key generation
	grp     *cyclic.Group
	userReg globals.UserRegistry

	// Inputs: user IDs and salts (required for key generation)
	users []*id.User
	salts [][]byte

	// Output: keys
	keysA *cyclic.IntBuffer
	keysB *cyclic.IntBuffer
}

// LinkStream This Link doesn't conform to the Stream interface because KeygenSubStream
// isn't meant for use alone in a graph
// For salts and users: the slices don't have to point to valid underlying data
// at Link time, but they should represent an area that'll be filled with valid
// data or space for data when the cryptop runs
func (k *KeygenSubStream) LinkStream(grp *cyclic.Group,
	userReg globals.UserRegistry, inSalts [][]byte, inUsers []*id.User,
	outKeysA, outKeysB *cyclic.IntBuffer) {
	k.grp = grp
	k.userReg = userReg
	k.salts = inSalts
	k.users = inUsers
	k.keysA = outKeysA
	k.keysB = outKeysB
}

//Returns the substream, used to return an embedded struct off an interface
func (k *KeygenSubStream) GetKeygenSubStream() *KeygenSubStream {
	return k
}

// *KeygenSubStream conforms to this interface, so pass the embedded substream
// struct to this module when you're using it
type KeygenSubStreamInterface interface {
	GetKeygenSubStream() *KeygenSubStream
}

var Keygen = services.Module{
	Adapt: func(s services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		streamInterface, ok := s.(KeygenSubStreamInterface)
		keygen, ok2 := cryptop.(cryptops.KeygenPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		kss := streamInterface.GetKeygenSubStream()

		hash, err := blake2b.New256(nil)

		if err != nil {
			jww.FATAL.Panicf("Could not get blake2b hash: %s", err.Error())
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			user, err := kss.userReg.GetUser(kss.users[i])
			if err != nil {
				return err
			}
			//fixme: figure out why this only works when using a temp variable
			tmp := kss.grp.NewInt(1)
			keygen(kss.grp, kss.salts[i], user.BaseKey, tmp)
			kss.grp.Set(kss.keysA.Get(i), tmp)

			hash.Reset()
			hash.Write(kss.salts[i])

			keygen(kss.grp, hash.Sum(nil), user.BaseKey, tmp)
			kss.grp.Set(kss.keysB.Get(i), tmp)

		}

		return nil
	},
	Cryptop:    cryptops.Keygen,
	InputSize:  services.AutoInputSize,
	Name:       "Keygen",
	NumThreads: services.AutoNumThreads,
}
