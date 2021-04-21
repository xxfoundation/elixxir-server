///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles the high level storage API.
// This layer merges the business logic layer and the database layer

package storage

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/nonce"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Storage API for the storage layer
type Storage struct {
	// Stored database interface
	database
}

// NewStorage Create a new Storage object wrapping a database interface
// Returns a Storage object, close function, and error
func NewStorage(username, password, dbName, address, port string, devMode bool) (*Storage, error) {
	db, err := newDatabase(username, password, dbName, address, port, devMode)
	storage := &Storage{db}
	return storage, err
}

func (c *Client) GetId() (*id.ID, error) {
	return id.Unmarshal(c.Id)
}

func (c *Client) GetBaseKey(grp *cyclic.Group) *cyclic.Int {
	return grp.NewIntFromBytes(c.BaseKey)
}

func (c *Client) GetPublicKey() (*rsa.PublicKey, error) {
	return rsa.LoadPublicKeyFromPem(c.PublicKey)
}

func (c *Client) GetNonce() nonce.Nonce {
	n := nonce.Nonce{
		GenTime:    c.NonceTimestamp,
		ExpiryTime: c.NonceTimestamp.Add(nonce.RegistrationTTL * time.Second),
		TTL:        nonce.RegistrationTTL * time.Second,
	}
	copy(n.Value[:], c.Nonce)
	return n
}
