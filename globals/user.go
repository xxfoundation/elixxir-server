////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally initiated UserRegistry
var Users = newUserRegistry()

// Number of hard-coded users to create
var NUM_DEMO_USERS = int(10)

// Globally initiated User ID counter
var idCounter = uint64(1)

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(address string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) (user *User, ok bool)
	GetNickList() (ids []uint64, nicks []string)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(huid uint64) (uint64, bool)
}

// Struct implementing the UserRegistry Interface with an underlying Map
type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	userLookup     map[uint64]uint64
}

type ForwardKey struct {
	BaseKey      *cyclic.Int
	RecursiveKey *cyclic.Int
}

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
	ID            uint64
	HUID          uint64
	Address       string
	Nick          string
	Transmission  ForwardKey
	Reception     ForwardKey
	PublicKey     *cyclic.Int
	MessageBuffer chan *pb.CmixMessage
}

// Deep Copy creates a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	nu := new(User)
	nu.ID = u.ID
	nu.HUID = u.HUID
	nu.Address = u.Address
	nu.Nick = u.Nick
	nu.Transmission = *u.Transmission.DeepCopy()
	nu.Reception = *u.Reception.DeepCopy()
	nu.PublicKey = cyclic.NewInt(0).Set(u.PublicKey)
	nu.MessageBuffer = u.MessageBuffer
	return nu
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(address string) *User {
	idCounter++
	usr := new(User)
	h := sha256.New()
	i := idCounter - 1
	trans := new(ForwardKey)
	recept := new(ForwardKey)

	// Generate user parameters
	usr.ID = uint64(i)
	usr.HUID = uint64(i + 10)
	h.Write([]byte(string(20000 + i)))
	trans.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h.Write([]byte(string(30000 + i)))
	trans.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h.Write([]byte(string(40000 + i)))
	recept.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
	h.Write([]byte(string(50000 + i)))
	recept.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
	usr.Reception = *recept
	usr.Transmission = *trans
	usr.PublicKey = cyclic.NewMaxInt()
	usr.MessageBuffer = make(chan *pb.CmixMessage, 100)
	return usr
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id uint64) {
	// If key does not exist, do nothing
	delete(m.userCollection, id)
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id uint64) (*User, bool) {
	var u *User
	u, ok := m.userCollection[id]
	user := u.DeepCopy()
	return user, ok
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.ID] = user
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

// GetNickList returns a slice of all the user IDs and a slice of the
// corresponding nicknames of all the users in a user map.
func (m *UserMap) GetNickList() (ids []uint64, nicks []string) {

	userCount := m.CountUsers()

	nicks = make([]string, 0, userCount)
	ids = make([]uint64, 0, userCount)
	for _, user := range m.userCollection {
		if user != nil {
			nicks = append(nicks, user.Nick)
			ids = append(ids, user.ID)
		} else {
			jwalterweatherman.FATAL.Panicf("A user was nil.")
		}
	}

	return ids, nicks
}

// LookupUser takes a hashed registration code and returns the corresponding
// User ID if it is found.
func (m *UserMap) LookupUser(huid uint64) (uint64, bool) {
	uid, ok := m.userLookup[huid]
	return uid, ok
}
