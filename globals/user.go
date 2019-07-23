////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

const MaxSalts = 300

var errNonexistantUser = "user %v not found in user registry"
var errTooManySalts = "user %v must rekey, has stored too many salts"
var ErrSaltIncorrectLength = errors.New("salt of incorrect length, must be 256 bits")
var ErrUserIDTooShort = errors.New("User id length too short")

// Globally initiated User ID counter
var idCounter = uint64(1)

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(grp *cyclic.Group) *User
	DeleteUser(id *id.User)
	GetUser(id *id.User) (user *User, err error)
	GetUserByNonce(nonce nonce.Nonce) (user *User, err error)
	UpsertUser(user *User)
	CountUsers() int
	InsertSalt(user *id.User, salt []byte) error
}

// Structure implementing the UserRegistry Interface with an underlying sync.Map
type UserMap sync.Map

// Structure representing a User in the system
type User struct {
	ID        *id.User
	HUID      []byte
	BaseKey   *cyclic.Int
	PublicKey *signature.DSAPublicKey
	Nonce     nonce.Nonce

	salts [][]byte
	sync.Mutex
}

// DeepCopy creates a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	newUser := new(User)
	newUser.ID = u.ID
	newUser.BaseKey = u.BaseKey.DeepCopy()

	if u.PublicKey != nil {
		params := u.PublicKey.GetParams()
		newUser.PublicKey = signature.ReconstructPublicKey(signature.
			CustomDSAParams(params.GetP(), params.GetQ(),
				params.GetG()), u.PublicKey.GetKey())
	}

	newUser.Nonce = nonce.Nonce{
		GenTime:    u.Nonce.GenTime,
		ExpiryTime: u.Nonce.ExpiryTime,
		TTL:        u.Nonce.TTL,
	}
	copy(u.Nonce.Bytes(), newUser.Nonce.Bytes())
	return newUser
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(grp *cyclic.Group) *User {
	idCounter++
	usr := new(User)
	h := sha256.New()
	i := idCounter - 1

	// Generate user parameters
	usr.ID = id.NewUserFromUints(&[4]uint64{0, 0, 0, i})

	h.Reset()
	h.Write([]byte(string(40000 + i)))
	usr.BaseKey = grp.NewIntFromBytes(h.Sum(nil))

	usr.PublicKey = signature.ReconstructPublicKey(
		signature.CustomDSAParams(
			large.NewInt(0), large.NewInt(0), large.NewInt(0),
		),
		large.NewInt(0),
	)

	usr.Nonce = *new(nonce.Nonce)

	return usr
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserMap) InsertSalt(id *id.User, salt []byte) error {
	// If the number of salts for the given UserId
	// is greater than the maximum allowed, then reject

	userFace, ok := (*sync.Map)(m).Load(*id)
	if !ok {
		return errors.New(fmt.Sprintf(errNonexistantUser, id))
	}

	user := userFace.(*User)
	user.Lock()
	defer user.Unlock()

	if len(user.salts) >= MaxSalts {
		jww.ERROR.Printf("Unable to insert salt: Too many salts have already"+
			" been used for User %q", *id)
		return errors.New(fmt.Sprintf(errTooManySalts, id))
	}

	// Insert salt into the collection
	user.salts = append(user.salts, salt)
	return nil
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id *id.User) {
	// If key does not exist, do nothing
	(*sync.Map)(m).Delete(*id)
}

// GetUser returns a user with the given ID from userCollection
func (m *UserMap) GetUser(id *id.User) (*User, error) {
	var err error
	var userCopy *User

	u, ok := (*sync.Map)(m).Load(*id)
	if !ok {
		err = errors.New(fmt.Sprintf(errNonexistantUser, id))
	} else {
		user := u.(*User)
		user.Lock()
		userCopy = user.DeepCopy()
		user.Unlock()
	}
	return userCopy, err
}

// GetUser returns a user with a matching nonce from userCollection
func (m *UserMap) GetUserByNonce(nonce nonce.Nonce) (user *User, err error) {
	var u *User
	ok := false

	// Iterate over the map to find user with matching nonce

	(*sync.Map)(m).Range(
		func(key, value interface{}) bool {
			uRtn := value.(*User)
			uRtn.Lock()
			if bytes.Equal(uRtn.Nonce.Bytes(), nonce.Bytes()) {
				ok = true
				u = uRtn
			}
			uRtn.Unlock()
			return !ok
		},
	)

	if !ok {
		err = errors.New("unable to lookup user by nonce in ram user registry")
	} else {
		user = u.DeepCopy()
	}
	return
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	(*sync.Map)(m).Store(*(user.ID), user)
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	numUser := 0

	(*sync.Map)(m).Range(
		func(key, value interface{}) bool {
			numUser++
			return true
		},
	)

	return numUser
}
