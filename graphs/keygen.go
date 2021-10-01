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
	goHash "hash"
	"strings"
)

// Module that implements Keygen, along with helper methods
type KeygenSubStream struct {
	// Server state that's needed for key generation
	Grp         *cyclic.Group
	NodeSecrets *storage.NodeSecretManager
	// todo: remove storage once databaseless implementation is fully tested
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
	inSalts [][]byte, inKMACS [][][]byte, inUsers []*id.ID,
	outKeysA, outKeysB *cyclic.IntBuffer, reporter *round.ClientReport,
	roundID id.Round, batchSize uint32, nodeSecrets *storage.NodeSecretManager) {
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
	k.NodeSecrets = nodeSecrets
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

		saltHash, err := blake2b.New256(nil)

		if err != nil {
			jww.FATAL.Panicf("Could not get blake2b hash: %s",
				err.Error())
		}

		kmacHash, err := hash.NewCMixHash()

		if err != nil {
			jww.FATAL.Panicf("Could not get CMIX hash: %s", err.Error())
		}

		nodeSecretHash, err := hash.NewCMixHash()
		if err != nil {
			jww.FATAL.Panicf("Could not get node secret hash: %s", err.Error())
		}

		kss.userErrors.InitErrorChan(kss.RoundId, kss.batchSize)

		for i := chunk.Begin(); i < chunk.End() && i < kss.batchSize; i++ {
			if kss.users[i].Cmp(&id.ID{}) {
				continue
			}
			// Retrieve the node secret
			// todo: KeyID will not be hardcoded once multiple rotating
			//  secrets is supported.
			nodeSecret, err := kss.NodeSecrets.GetSecret(0)
			if err != nil {
				if strings.HasSuffix(err.Error(), storage.NoSecretExistsError) {
					jww.INFO.Printf("No secret for key ID %d with user %v found for slot %d",
						0, kss.users[i], i)
					kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
					kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
					errMsg := fmt.Sprintf("%s [%v] in storage:%v",
						services.SecretNotFound, kss.users[i], err)
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

			// Generate node key
			nodeSecretHash.Reset()
			nodeSecretHash.Write(kss.users[i].Bytes())
			nodeSecretHash.Write(nodeSecret.Bytes())
			clientKeyBytes := nodeSecretHash.Sum(nil)

			newRegPathSuccess := false

			clientKey := kss.Grp.NewIntFromBytes(clientKeyBytes)
			if len(kss.kmacs[i]) != 0 {
				if cmix.VerifyKMAC(kss.kmacs[i][0], kss.salts[i], clientKey, kss.RoundId, kmacHash) {
					keygen(kss.Grp, kss.salts[i], kss.RoundId,
						clientKey, kss.KeysA.Get(i))

					saltHash.Reset()
					saltHash.Write(kss.salts[i])

					keygen(kss.Grp, saltHash.Sum(nil), kss.RoundId,
						clientKey, kss.KeysB.Get(i))
					newRegPathSuccess = true
				} else {
					jww.INFO.Printf("KMAC ERR: %v not the same as %v",
						kss.kmacs[i][0], cmix.GenerateKMAC(kss.salts[i],
							clientKey, kss.RoundId, kmacHash))
				}
			}

			// todo: this remains for backwards compatibility purposes
			//  while testing new registration path. Rip this out when
			//  RequestClientKey has been properly tested
			success, isContinue := false, false
			if !newRegPathSuccess {
				success, isContinue, err = oldRegPath(kss, kmacHash, saltHash, keygen, i)
				if err != nil {
					return err
				}
			}
			if isContinue {
				continue
			}

			//pop the used KMAC
			kss.kmacs[i] = kss.kmacs[i][1:]

			if !success && !newRegPathSuccess {
				kss.Grp.SetUint64(kss.KeysA.Get(i), 1)
				kss.Grp.SetUint64(kss.KeysB.Get(i), 1)
				jww.DEBUG.Printf("User: %#v", kss.users[i])
				jww.DEBUG.Printf("KMACS: %#v", kss.kmacs[i])
				jww.DEBUG.Printf("User %v on slot %v could not be "+
					"validated", kss.users[i], i)
				errMsg := fmt.Sprintf("%s. UserID [%v] failed on "+
					"slot %d", services.InvalidMAC, kss.users[i], i)
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

// todo: this remains for backwards compatibility purposes
//  while testing new registration path. Rip this out when
//  RequestClientKey has been properly tested
func oldRegPath(kss *KeygenSubStream, kmacHash, saltHash goHash.Hash,
	keygen cryptops.KeygenPrototype, index uint32) (success, isContinue bool, err error) {
	jww.WARN.Printf("DEPRECATED: Attempting old keyGen path with user %v", kss.users[index])

	user, err := kss.storage.GetClient(kss.users[index])
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			jww.INFO.Printf("No user %s found for slot %d",
				kss.users[index], index)
			kss.Grp.SetUint64(kss.KeysA.Get(index), 1)
			kss.Grp.SetUint64(kss.KeysB.Get(index), 1)
			errMsg := fmt.Sprintf("%s [%v] in storage:%v",
				services.UserNotFound, kss.users[index], err)
			clientError := &pb.ClientError{
				ClientId: kss.users[index].Bytes(),
				Error:    errMsg,
			}

			err = kss.userErrors.Send(kss.RoundId, clientError)
			if err != nil {
				return false, false, err
			}

		}
		return false, true, nil
	}

	if user.IsRegistered && len(kss.kmacs[index]) != 0 {
		clientBaseKey := user.GetDhKey(kss.Grp)
		//check the KMAC
		if cmix.VerifyKMAC(kss.kmacs[index][0], kss.salts[index], clientBaseKey,
			kss.RoundId, kmacHash) {
			keygen(kss.Grp, kss.salts[index], kss.RoundId,
				clientBaseKey, kss.KeysA.Get(index))

			saltHash.Reset()
			saltHash.Write(kss.salts[index])

			keygen(kss.Grp, saltHash.Sum(nil), kss.RoundId,
				clientBaseKey, kss.KeysB.Get(index))
			success = true
		} else {
			jww.INFO.Printf("KMAC ERR: %v not the same as %v",
				kss.kmacs[index][0], cmix.GenerateKMAC(kss.salts[index],
					clientBaseKey, kss.RoundId, kmacHash))
		}
	}

	return success, false, nil

}
