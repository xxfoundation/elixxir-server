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
	GetUser(id uint64) (user *User, ok bool)
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
	RecursiveKey   *cyclic.Int
}

func (fk *ForwardKey) DeepCopy()(*ForwardKey){

	if fk == nil{
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

func (u *User) DeepCopy()(*User){

	if u == nil{
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
	m.idCounter++
	return &User{Id: m.idCounter - 1, Address: address,
		Transmission: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			RecursiveKey:   cyclic.NewMaxInt()},
		Reception: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			RecursiveKey:   cyclic.NewMaxInt()},
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
func (m *UserMap) GetUser(id uint64) (user *User, ok bool) {
	var u *User
	u, ok = m.userCollection[id]
	user = u.DeepCopy()
	return
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
