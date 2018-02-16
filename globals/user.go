package globals

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally initiated UserRegistry
var Users = newUserRegistry()

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(address string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) *User
	UpsertUser(user *User)
	CountUsers() int
}

// Struct implementing the UserRegistry Interface with an underlying Map
type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Increments sequentially for User.id values
	idCounter uint64
}

// Creates a new UserRegistry interface
func newUserRegistry() UserRegistry {
	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: make(map[uint64]*User), idCounter: 0})
}

type ForwardKey struct {
	BaseKey        *cyclic.Int
	PartialBaseKey *cyclic.Int
	RecursiveKey   *cyclic.Int
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

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(address string) *User {
	m.idCounter++
	return &User{Id: m.idCounter - 1, Address: address,
		Transmission: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		Reception: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		PublicKey: cyclic.NewMaxInt(),
	}
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id uint64) {
	// If key does not exist, do nothing
	delete(m.userCollection, id)
}

// GetUser returns a user with the given ID from userCollection.
func (m *UserMap) GetUser(id uint64) *User {
	// If key does not exist, return nil
	return m.userCollection[id]
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.Id] = user
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}
