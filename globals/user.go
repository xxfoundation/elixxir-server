////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally initiated UserRegistry
var Users = newUserRegistry()

// Globally initiated User Id counter
var idCounter = uint64(1)

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(address string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
}

// Struct implementing the UserRegistry Interface with an underlying Map
type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
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
	Id      uint64
	Address string

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

	nu.Id = u.Id
	nu.Address = u.Address

	nu.Transmission = *u.Transmission.DeepCopy()

	nu.Reception = *u.Reception.DeepCopy()

	nu.PublicKey = cyclic.NewInt(0).Set(u.PublicKey)

	nu.MessageBuffer = u.MessageBuffer

	return nu
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(address string) *User {
	idCounter++
	return &User{Id: idCounter - 1, Address: address,
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
	}
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
	m.userCollection[user.Id] = user
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}
