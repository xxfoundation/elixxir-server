////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"sync"
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
	NewUser(address string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) (user *User, err error)
	GetNickList() (ids []uint64, nicks []string)
	UpsertUser(user *User)
	CountUsers() int
	InsertSalt(userId uint64, salt []byte) bool
}

// Struct implementing the UserRegistry Interface with an underlying Map
type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Map acting as the Salt table, containing UserID -> List of 256-bit salts
	saltCollection map[uint64][][]byte
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
	ID            uint64
	HUID          uint64
	Address       string
	Nick          string
	Transmission  ForwardKey
	Reception     ForwardKey
	PublicKey     *cyclic.Int
	MessageBuffer chan *pb.CmixMessage
}

// DeepCopy creates a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	newUser := new(User)
	newUser.ID = u.ID
	newUser.Address = u.Address
	newUser.Nick = u.Nick
	newUser.Transmission = *u.Transmission.DeepCopy()
	newUser.Reception = *u.Reception.DeepCopy()
	newUser.PublicKey = cyclic.NewInt(0).Set(u.PublicKey)
	newUser.MessageBuffer = u.MessageBuffer
	return newUser
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
	usr.PublicKey = cyclic.NewMaxInt()
	usr.MessageBuffer = make(chan *pb.CmixMessage, 50000)
	return usr
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserMap) InsertSalt(userId uint64, salt []byte) bool {
	// If the number of salts for the given UserId
	// is greater than the maximum allowed, then reject
	maxSalts := 300
	if len(m.saltCollection[userId]) > maxSalts {
		jww.ERROR.Printf("Unable to insert salt: Too many salts have already"+
			" been used for User %d", userId)
		return false
	}

	// Insert salt into the collection
	m.saltCollection[userId] = append(m.saltCollection[userId], salt)
	return true
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id uint64) {
	// If key does not exist, do nothing
	m.collectionLock.Lock()
	delete(m.userCollection, id)
	m.collectionLock.Unlock()
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id uint64) (*User, error) {
	var u *User
	var err error
	m.collectionLock.Lock()
	u, ok := m.userCollection[id]
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
	m.userCollection[user.ID] = user
	m.collectionLock.Unlock()
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
	m.collectionLock.Lock()
	for _, user := range m.userCollection {
		if user != nil {
			nicks = append(nicks, user.Nick)
			ids = append(ids, user.ID)
		} else {
			jww.FATAL.Panicf("A user was nil.")
		}
	}
	m.collectionLock.Unlock()

	return ids, nicks
}
