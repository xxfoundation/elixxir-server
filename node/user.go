package node

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// userRegistry is a map which acts as a global registry that holds all users.
var userRegistry map[uint64]User

// idCounter is a sequential counter for user.Id values.
var idCounter uint64 = 0

type ForwardKey struct {
	BaseKey        *cyclic.Int
	PartialBaseKey *cyclic.Int
	RecursiveKey   *cyclic.Int
}

type User struct {
	Id      uint64
	Address string

	Transmission ForwardKey
	Reception    ForwardKey

	PublicKey *cyclic.Int
}

// InitUserRegistry initializes the userRegistry map.
func InitUserRegistry() {
	userRegistry = make(map[uint64]User)
}

// NewUser creates a new User object with default fields and given address.
func NewUser(address string) User {
	idCounter++
	return User{Id: idCounter - 1, Address: address,
		Transmission: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		Reception: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		PublicKey: cyclic.NewMaxInt(),
	}
}

// DeleteUser deletes a user with the given ID from userRegistry.
func DeleteUser(id uint64) {
	// If key does not exist, do nothing
	delete(userRegistry, id)
}

// GetUser returns a user with the given ID from userRegistry.
func GetUser(id uint64) User {
	// If key does not exist, return nil
	return userRegistry[id]
}

// UpsertUser inserts given user into userRegistry or update the user if it
// already exists (Upsert operation).
func UpsertUser(user User) {
	userRegistry[user.Id] = user
}

// CountUsers returns a count of the users in userRegistry
func CountUsers() int {
	return len(userRegistry)
}
