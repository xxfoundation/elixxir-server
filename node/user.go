package node

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Map which acts as a global registry that holds all users
var userRegistry map[uint64]User

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

func NewUser(id uint64, address string) User {
	return User{Id: id, Address: address,
		Transmission: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		Reception: ForwardKey{BaseKey: cyclic.NewMaxInt(),
			PartialBaseKey: cyclic.NewMaxInt(), RecursiveKey: cyclic.NewMaxInt()},
		PublicKey: cyclic.NewMaxInt(),
	}
}

// Deletes a user with the given ID from userRegistry
func DeleteUser(id uint64) {
	// If key does not exist, do nothing
	delete(userRegistry, id)
}

// Returns a user with the given ID from userRegistry
func GetUser(id uint64) User {
	// If key does not exist, return nil
	return userRegistry[id]
}

// Insert given user into userRegistry or update the user if it already exists (Upsert operation)
func UpsertUser(user User) {
	userRegistry[user.Id] = user
}

// Returns a count of the users in userRegistry
func CountUsers() int {
	return len(userRegistry)
}
