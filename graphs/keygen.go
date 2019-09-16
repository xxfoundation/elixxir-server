package graphs

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
)

// Module that implements Keygen, along with helper methods
type KeygenSubStream struct {
	// Server state that's needed for key generation
	Grp     *cyclic.Group
	userReg globals.UserRegistry

	// Inputs: user IDs and salts (required for key generation)
	users []*id.User
	salts [][]byte
	kmacs [][][]byte

	// Output: keys
	KeysA *cyclic.IntBuffer
	KeysB *cyclic.IntBuffer
}

// LinkStream This Link doesn't conform to the Stream interface because KeygenSubStream
// isn't meant for use alone in a graph
// For salts and users: the slices don't have to point to valid underlying data
// at Link time, but they should represent an area that'll be filled with valid
// data or space for data when the cryptop runs
func (k *KeygenSubStream) LinkStream(grp *cyclic.Group,
	userReg globals.UserRegistry, inSalts [][]byte, imKMACS [][][]byte, inUsers []*id.User,
	outKeysA, outKeysB *cyclic.IntBuffer) {
	k.Grp = grp
	k.userReg = userReg
	k.salts = inSalts
	k.users = inUsers
	k.kmacs = imKMACS
	k.KeysA = outKeysA
	k.KeysB = outKeysB
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

		salthash, err := blake2b.New256(nil)

		if err != nil {
			jww.FATAL.Panicf("Could not get blake2b hash: %s", err.Error())
		}

		kmacHash, err := hash.NewCMixHash()

		if err != nil {
			jww.FATAL.Panicf("Could not get CMIX hash: %s", err.Error())
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			user, err := kss.userReg.GetUser(kss.users[i])

			if err != nil {
				if err.Error() == "pg: no rows in result set" ||
					err == globals.ErrNonexistantUser {
					jww.INFO.Printf("No user found for slot %d", i)
					kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
					kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
					return nil
				}
				return err
			}
			//fixme: figure out why this only works when using a temp variable
			tmp := kss.Grp.NewInt(1)

			success := false
			jww.DEBUG.Printf("kss: %v", kss.users)
			if user.IsRegistered && len(kss.kmacs[i]) != 0 {
				//check the KMAC
				if cmix.VerifyKMAC(kss.kmacs[i][0], kss.salts[i], user.BaseKey, kmacHash) {
					keygen(kss.Grp, kss.salts[i], user.BaseKey, tmp)
					kss.Grp.Set(kss.KeysA.Get(i), tmp)

					salthash.Reset()
					salthash.Write(kss.salts[i])

					keygen(kss.Grp, salthash.Sum(nil), user.BaseKey, tmp)
					kss.Grp.Set(kss.KeysB.Get(i), tmp)
					success = true
				} else {
					jww.INFO.Printf("KMAC ERR: %v not the same as %v", kss.kmacs[i][0], cmix.GenerateKMAC(kss.salts[i], user.BaseKey, kmacHash))
				}
				//pop the used KMAC
				kss.kmacs[i] = kss.kmacs[i][1:]
			}

			if !success {
				kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
				kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
				jww.INFO.Printf("User: %#v", user)
				jww.INFO.Printf("KMACS: %#v", kss.kmacs[i])
				jww.INFO.Printf("User %v on slot %v could not be validated",
					user.ID, i)
			}

		}

		return nil
	},
	Cryptop:    cryptops.Keygen,
	InputSize:  services.AutoInputSize,
	Name:       "Keygen",
	NumThreads: services.AutoNumThreads,
}
