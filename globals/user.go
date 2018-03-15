////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"github.com/spf13/jwalterweatherman"
	"crypto/sha256"
	"fmt"
)

// Globally initiated UserRegistry
var Users = newUserRegistry()

// Number of hard-coded users to create
var NUM_DEMO_USERS = int(10)

// Globally initiated User UID counter
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
	userLookup map[uint64]uint64
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
	UID     uint64
	HUID	uint64
	Address string
	Nick    string

	Transmission ForwardKey
	Reception    ForwardKey

	PublicKey     *cyclic.Int
	MessageBuffer chan *pb.CmixMessage
}

func (u *User) DeepCopy() *User {

	if u == nil {
		return nil
	}

	nu := new(User)

	nu.UID = u.UID
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
	/*return &User{UID: idCounter - 1, Address: address, Nick: "",
		// TODO: each user should have unique base and secret keys
		Transmission: ForwardKey{BaseKey: cyclic.NewIntFromString(
			"c1248f42f8127999e07c657896a26b56fd9a499c6199e1265053132451128f52", 16),
			RecursiveKey: cyclic.NewIntFromString(
				"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251", 16)},
		Reception: ForwardKey{BaseKey: cyclic.NewIntFromString(
			"83120e7bfaba497f8e2c95457a28006f73ff4ec75d3ad91d27bf7ce8f04e772c", 16),
			RecursiveKey: cyclic.NewIntFromString(
				"979e574166ef0cd06d34e3260fe09512b69af6a414cf481770600d9c7447837b", 16)},
		PublicKey:     cyclic.NewMaxInt(),
		MessageBuffer: make(chan *pb.CmixMessage, 100),
	}*/
	usr := new(User)
	h := sha256.New()
	i := idCounter -1
	trans := new(ForwardKey)
	recept := new(ForwardKey)
	// Generate user parameters
	usr.UID = uint64(i)
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
	fmt.Printf("Forward Keys: %v\n%v", usr.Transmission.RecursiveKey.Text(16),
		usr.Transmission.BaseKey.Text(16))
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
	m.userCollection[user.UID] = user
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

func (m *UserMap) GetNickList() (ids []uint64, nicks []string) {

	userCount := m.CountUsers()

	nicks = make([]string, 0, userCount)
	ids = make([]uint64, 0, userCount)
	for _, user := range m.userCollection {
		if user != nil {
			nicks = append(nicks, user.Nick)
			ids = append(ids, user.UID)
		} else {
			jwalterweatherman.FATAL.Panicf("A user was nil.")
		}
	}

	return ids, nicks
}

func (m *UserMap) LookupUser(huid uint64) (uint64, bool) {
	return m.userLookup[huid], true
}