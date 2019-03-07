////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/dsa"
	"crypto/sha256"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"math/big"
	"sync"
	"time"
)

// Globally initiated UserRegistry
var Users UserRegistry

// Number of hard-coded users to create
var NUM_DEMO_USERS = int(30)
var NUM_DEMO_CHANNELS = int(10)

// Globally initiated User ID counter
var idCounter = uint64(1)

// Interface for User Registry operations
type UserRegistry interface {
	NewUser() *User
	DeleteUser(id *id.User)
	GetUser(id *id.User) (user *User, err error)
	UpsertUser(user *User)
	CountUsers() int
	InsertSalt(user *id.User, salt []byte) bool
}

// Struct implementing the UserRegistry Interface with an underlying Map
type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[id.User]*User
	// Map acting as the Salt table, containing User -> List of 256-bit salts
	saltCollection map[id.User][][]byte
	// Lock for thread safety
	collectionLock *sync.Mutex
}

type ForwardKey struct {
	BaseKey      *cyclic.Int
	RecursiveKey *cyclic.Int
}

// DeepCopy creates a deep copy of a ForwardKey struct and returns a pointer
// to the new copy
func (fk *ForwardKey) DeepCopy() *ForwardKey {

	if fk == nil {
		return nil
	}
	nfk := ForwardKey{
		cyclic.NewInt(0),
		cyclic.NewInt(0),
	}
	nfk.BaseKey.Set(fk.BaseKey)
	nfk.RecursiveKey.Set(fk.RecursiveKey)
	return &nfk
}

// Struct representing a User in the system
type User struct {
	ID             *id.User
	HUID           []byte
	Transmission   ForwardKey
	Reception      ForwardKey
	PublicKey      *dsa.PublicKey
	Nonce          []byte
	NonceTimestamp time.Time
}

// DeepCopy creates a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	newUser := new(User)
	newUser.ID = u.ID
	newUser.Transmission = *u.Transmission.DeepCopy()
	newUser.Reception = *u.Reception.DeepCopy()

	newUser.PublicKey = new(dsa.PublicKey)
	newUser.PublicKey.Y = u.PublicKey.Y
	newUser.PublicKey.P = u.PublicKey.P
	newUser.PublicKey.Q = u.PublicKey.Q
	newUser.PublicKey.G = u.PublicKey.G

	newUser.Nonce = make([]byte, len(u.Nonce))
	copy(newUser.Nonce, u.Nonce)
	newUser.NonceTimestamp = u.NonceTimestamp
	return newUser
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser() *User {
	idCounter++
	usr := new(User)
	h := sha256.New()
	i := idCounter - 1
	trans := new(ForwardKey)
	recept := new(ForwardKey)

	// Generate user parameters
	usr.ID = new(id.User).SetUints(&[4]uint64{0, 0, 0, i})
	h.Write([]byte(string(20000 + i)))
	trans.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h = sha256.New()
	h.Write([]byte(string(30000 + i)))
	trans.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h = sha256.New()
	h.Write([]byte(string(40000 + i)))
	recept.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h = sha256.New()
	h.Write([]byte(string(50000 + i)))
	recept.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
	usr.Reception = *recept
	usr.Transmission = *trans

	usr.PublicKey = new(dsa.PublicKey)
	usr.PublicKey.Y = new(big.Int)
	usr.PublicKey.P = new(big.Int)
	usr.PublicKey.Q = new(big.Int)
	usr.PublicKey.G = new(big.Int)

	usr.Nonce = make([]byte, 0)
	usr.NonceTimestamp = *new(time.Time)
	return usr
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserMap) InsertSalt(user *id.User, salt []byte) bool {
	// If the number of salts for the given UserId
	// is greater than the maximum allowed, then reject
	maxSalts := 300
	if len(m.saltCollection[*user]) > maxSalts {
		jww.ERROR.Printf("Unable to insert salt: Too many salts have already"+
			" been used for User %q", *user)
		return false
	}

	// Insert salt into the collection
	m.saltCollection[*user] = append(m.saltCollection[*user], salt)
	return true
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(user *id.User) {
	// If key does not exist, do nothing
	m.collectionLock.Lock()
	delete(m.userCollection, *user)
	m.collectionLock.Unlock()
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(user *id.User) (*User, error) {
	var u *User
	var err error
	m.collectionLock.Lock()
	u, ok := m.userCollection[*user]
	m.collectionLock.Unlock()

	if !ok {
		err = errors.New("unable to lookup user in ram user registry")
	} else {
		u = u.DeepCopy()
	}
	return u, err
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.collectionLock.Lock()
	m.userCollection[*user.ID] = user
	m.collectionLock.Unlock()
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}
