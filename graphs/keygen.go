///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"gorm.io/gorm"
)

// Module that implements Keygen, along with helper methods
type KeygenSubStream struct {
	// Server state that's needed for key generation
	Grp     *cyclic.Group
	storage *storage.Storage

	// Inputs: user IDs and salts (required for key generation)
	users []*id.ID
	salts [][]byte
	kmacs [][][]byte

	// Output: keys
	KeysA *cyclic.IntBuffer
	KeysB *cyclic.IntBuffer

	userErrors *round.ClientReport
	RoundId    id.Round
	batchSize  uint32
}

// LinkStream This Link doesn't conform to the Stream interface because KeygenSubStream
// isn't meant for use alone in a graph
// For salts and users: the slices don't have to point to valid underlying data
// at Link time, but they should represent an area that'll be filled with valid
// data or space for data when the cryptop runs
func (k *KeygenSubStream) LinkStream(grp *cyclic.Group, storage *storage.Storage,
	inSalts [][]byte, inKMACS [][][]byte, inUsers []*id.ID, outKeysA,
	outKeysB *cyclic.IntBuffer, reporter *round.ClientReport, roundID id.Round,
	batchSize uint32) {
	k.Grp = grp
	k.storage = storage
	k.salts = inSalts
	k.users = inUsers
	k.kmacs = inKMACS
	k.KeysA = outKeysA
	k.KeysB = outKeysB
	k.userErrors = reporter
	k.RoundId = roundID
	k.batchSize = batchSize
}

//Returns the substream, used to return an embedded struct off an interface
func (k *KeygenSubStream) GetKeygenSubStream() *KeygenSubStream {
	return k
}

// KeygenSubStream conforms to this interface, so pass the embedded substream
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
			jww.FATAL.Panicf("Could not get blake2b hash: %s",
				err.Error())
		}

		kmacHash, err := hash.NewCMixHash()

		if err != nil {
			jww.FATAL.Panicf("Could not get CMIX hash: %s", err.Error())
		}

		kss.userErrors.InitErrorChan(kss.RoundId, kss.batchSize)

		for i := chunk.Begin(); i < chunk.End() && i < kss.batchSize; i++ {
			if kss.users[i].Cmp(&id.ID{}) {
				continue
			}

			user, err := kss.storage.GetClient(kss.users[i])

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					jww.INFO.Printf("No user %s found for slot %d",
						kss.users[i], i)
					kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
					kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
					errMsg := fmt.Sprintf("%s [%v] in storage:%v",
						services.UserNotFound, kss.users[i], err)
					clientError := &pb.ClientError{
						ClientId: kss.users[i].Bytes(),
						Error:    errMsg,
					}

					err = kss.userErrors.Send(kss.RoundId, clientError)
					if err != nil {
						return err
					}

				}
				continue
			}

			success := false
			if user.IsRegistered && len(kss.kmacs[i]) != 0 {
				clientBaseKey := user.GetDhKey(kss.Grp)
				//check the KMAC
				if cmix.VerifyKMAC(kss.kmacs[i][0], kss.salts[i], clientBaseKey,
					kss.RoundId, kmacHash) {
					keygen(kss.Grp, kss.salts[i], kss.RoundId,
						clientBaseKey, kss.KeysA.Get(i))

					salthash.Reset()
					salthash.Write(kss.salts[i])

					keygen(kss.Grp, salthash.Sum(nil), kss.RoundId,
						clientBaseKey, kss.KeysB.Get(i))
					success = true
				} else {
					jww.INFO.Printf("KMAC ERR: %v not the same as %v",
						kss.kmacs[i][0], cmix.GenerateKMAC(kss.salts[i],
							clientBaseKey, kss.RoundId, kmacHash))
				}
				//pop the used KMAC
				kss.kmacs[i] = kss.kmacs[i][1:]
			}

			if !success {
				kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
				kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
				jww.DEBUG.Printf("User: %#v", user)
				jww.DEBUG.Printf("KMACS: %#v", kss.kmacs[i])
				jww.DEBUG.Printf("User %v on slot %v could not be "+
					"validated", user.Id, i)
				errMsg := fmt.Sprintf("%s. UserID [%v] failed on "+
					"slot %d", services.InvalidMAC, user.Id, i)
				clientError := &pb.ClientError{
					ClientId: kss.users[i].Bytes(),
					Error:    errMsg,
				}

				err = kss.userErrors.Send(kss.RoundId, clientError)
				if err != nil {
					return err
				}

			}

		}

		return nil
	},
	Cryptop:    cryptops.Keygen,
	InputSize:  services.AutoInputSize,
	Name:       "Keygen",
	NumThreads: services.AutoNumThreads,
}
