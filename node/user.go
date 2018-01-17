package node

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Map which acts as a global registry that holds all users
var userRegistry map[uint64]User

type TransmissionKey struct {
	BaseKey      *cyclic.Int
	RecursiveKey *cyclic.Int
}

type User struct {
	id uint64

	send    TransmissionKey
	receive TransmissionKey

	address string

	publicKey  *cyclic.Int
	privateKey *cyclic.Int
}

// Deletes a user with the given ID from userRegistry
func DeleteUser(id uint64) {
	delete(userRegistry, id)
}

// Returns a user with the given ID from userRegistry
func GetUser(id uint64) User {
	return userRegistry[id]
}

// Insert given user into userRegistry or update the user if it already exists (Upsert operation)
func UpsertUser(user User) {
	userRegistry[user.id] = user
}

// Returns a count of the users in userRegistry
func CountUsers() int {
	return len(userRegistry)
}
