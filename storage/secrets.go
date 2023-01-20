////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/nike"
	"sync"
)

// Error constants
const (
	BadSecretSizeError  = "Secret exceeds secret size"
	ManagerFullError    = "Manager is full"
	NoSecretExistsError = "No secret exists for key ID %d"
)

// Secret manager constants
const (
	// SecretSize is the defined size fo a node secret in bytes.
	SecretSize = 32

	// MaxNodeSecrets is the maximum amount of node secrets that will be stored in
	// RAM.
	MaxNodeSecrets = 256
)

// Secret represents the data within a NodeSecret. This is defined as a
// 32 byte (256 bit) byte array.
type Secret [SecretSize]byte

// NodeSecret contains a Secret. This will be used for:
// client registration (io/registration.go), and realtime keygen (graphs/keygen.go).
type NodeSecret struct {
	Secret Secret
	// todo: create a way to clear out old secrets and rotate them with new ones
	//TimeCreated time.Time  // Left as a stub
}

// NodeSecretManager will manage and rotate node secrets for client
// registration.
// fixme: this is only partially implemented, will need to have
//
//	rotating secrets
type NodeSecretManager struct {
	secrets      map[int]*NodeSecret
	mux          sync.Mutex
	ephemeralKey nike.PrivateKey
	ephemeralPub nike.PublicKey
}

// NewNodeSecretManager is the constructor for a NodeSecretManager. This will
// initialize an empty map mapping keyIds to nodeSecrets with a maximum size
// of MaxNodeSecrets.
func NewNodeSecretManager() *NodeSecretManager {
	return &NodeSecretManager{
		secrets: make(map[int]*NodeSecret, MaxNodeSecrets),
	}
}

func (nsm *NodeSecretManager) UpsertEphemerals(pub nike.PublicKey, priv nike.PrivateKey) {
	nsm.ephemeralKey = priv
	nsm.ephemeralPub = pub
}

func (nsm *NodeSecretManager) GetEphemerals() (nike.PublicKey, nike.PrivateKey) {
	return nsm.ephemeralPub, nsm.ephemeralKey
}

// GetSecret retrieves the Secret data associated with the given key ID
// from the map.
func (nsm *NodeSecretManager) GetSecret(keyId int) (Secret, error) {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()
	val, ok := nsm.secrets[keyId]
	if !ok {
		return Secret{}, errors.Errorf(NoSecretExistsError, keyId)
	}

	return val.Secret, nil
}

// UpsertSecret inserts a node secret into the NodeSecretManager.
// It will overwrite the existing secret if one exists.
func (nsm *NodeSecretManager) UpsertSecret(keyId int, data []byte) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()
	if len(nsm.secrets) == MaxNodeSecrets {
		return errors.Errorf(ManagerFullError)
	}

	if len(data) > SecretSize {
		return errors.Errorf(BadSecretSizeError)
	}

	// Copy data into secret
	secret := Secret{}
	copy(secret[:], data)

	// Place secret in map
	nsm.secrets[keyId] = &NodeSecret{
		Secret: secret,
	}

	return nil
}

// getNodeSecret returns the entire NodeSecret object from the map.
// This function is meant to be called by the manager in its management thread.
func (nsm *NodeSecretManager) getNodeSecret(keyId int) (*NodeSecret, error) {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	val, ok := nsm.secrets[keyId]
	if !ok {
		return &NodeSecret{}, errors.Errorf(NoSecretExistsError, keyId)
	}

	return val, nil
}

// delete is a deletion operation. This function is meant to be called by
// the manager in its management thread.
func (nsm *NodeSecretManager) delete(keyId int) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.secrets[keyId]; !ok {
		return errors.Errorf(NoSecretExistsError, keyId)
	}

	delete(nsm.secrets, keyId)

	return nil
}

// fixme: some mechanism is needed to clear out old secrets once
//  introducing new ones to avoid a memory leak issue in
//  NodeSecretManager's map. This function is left here as a stub to be
//  implemented once a design is complete.
//func (nsm *NodeSecretManager) ClearOldSecrets() {
//	// todo: implement me
//}

// Bytes returns the NodeSecret as a byte slice.
func (s Secret) Bytes() []byte {
	return s[:]
}

// String returns the Secret as a base 64 encoded string. This functions
// satisfies the fmt.Stringer interface.
func (s Secret) String() string {
	return base64.StdEncoding.EncodeToString(s.Bytes())
}
