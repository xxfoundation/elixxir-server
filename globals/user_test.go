////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"bytes"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
// TODO: This test needs split up
func TestUserRegistry(t *testing.T) {
	InitCrypto()

	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.User]*User),
		collectionLock: &sync.Mutex{},
	})

	for i := 0; i < NUM_DEMO_USERS; i++ {
		u := Users.NewUser(Group)
		Users.UpsertUser(u)
	}

	// TESTS Start here
	test := 6
	pass := 0

	numUsers := Users.CountUsers()

	if numUsers != NUM_DEMO_USERS {
		t.Errorf("Count users is not working correctly. "+
			"Expected: %v Actual: %v", NUM_DEMO_USERS, numUsers)
	} else {
		pass++
	}

	id9 := id.NewUserFromUint(9, t)
	usr9, _ := Users.GetUser(id9)

	if usr9 == nil {
		t.Errorf("Error fetching user!")
	} else {
		pass++
	}

	getUser, err := Users.GetUser(usr9.ID)

	if (err != nil) || getUser.ID != usr9.ID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	usr3, _ := Users.GetUser(id.NewUserFromUint(3, t))
	usr5, _ := Users.GetUser(id.NewUserFromUint(5, t))

	if usr3.Transmission.BaseKey == nil {
		t.Errorf("Error Setting the Transmission Base Key")
	} else {
		pass++
	}

	if usr3.Reception.BaseKey == usr5.Reception.BaseKey {
		t.Errorf("Transmissions keys are the same and they should be different!")
	} else {
		pass++
	}

	Users.DeleteUser(usr9.ID)

	if Users.CountUsers() != NUM_DEMO_USERS-1 {
		t.Errorf("User has not been deleted correctly. "+
			"Expected # of users: %v Actual: %v", NUM_DEMO_USERS-1, Users.CountUsers())
	} else {
		pass++
	}

	if _, userExists := Users.GetUser(usr9.ID); userExists == nil {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	} else {
		pass++
	}

	println("User Test", pass, "out of", test, "tests passed.")
}

// Test happy path
func TestForwardKey_DeepCopy(t *testing.T) {
	InitCrypto()

	fk := ForwardKey{
		BaseKey:      Group.NewInt(10),
		RecursiveKey: Group.NewInt(15),
	}

	nk := fk.DeepCopy()

	if fk.RecursiveKey.Cmp(nk.RecursiveKey) != 0 {
		t.Errorf("FK Deepcopy: Failed to copy recursive key!")
	}
	if fk.BaseKey.Cmp(nk.BaseKey) != 0 {
		t.Errorf("FK Deepcopy: Failed to copy base key!")
	}
}

// Test nil path
func TestForwardKey_DeepCopyNil(t *testing.T) {
	var fk *ForwardKey = nil

	nk := fk.DeepCopy()

	if nk != nil {
		t.Errorf("FK Deepcopy: Expected nil copy!")
	}
}

// Test happy path
func TestUser_DeepCopy(t *testing.T) {
	InitCrypto()

	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.User]*User),
		collectionLock: &sync.Mutex{},
	})

	user := Users.NewUser(Group)
	user.Transmission.BaseKey = Group.NewInt(66)

	newUser := user.DeepCopy()
	if user.Transmission.BaseKey.Cmp(newUser.Transmission.BaseKey) != 0 {
		t.Errorf("User Deepcopy: Failed to copy keys!")
	}
}

// Test happy path and inserting too many salts
func TestUserMap_InsertSalt(t *testing.T) {
	Users := UserRegistry(&UserMap{
		saltCollection: make(map[id.User][][]byte),
	})
	// Insert like 300 salts, expect success
	for i := 0; i <= 300; i++ {
		if !Users.InsertSalt(id.NewUserFromUint(1, t), []byte("test")) {
			t.Errorf("InsertSalt: Expected success!")
		}
	}
	// Now we have exceeded the max number, expect failure
	if Users.InsertSalt(id.NewUserFromUint(1, t), []byte("test")) {
		t.Errorf("InsertSalt: Expected failure due to exceeding max count of" +
			" salts for one user!")
	}
}

// Test happy path
func TestUserMap_GetUserByNonce(t *testing.T) {
	InitCrypto()

	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.User]*User),
		collectionLock: &sync.Mutex{},
	})

	user := Users.NewUser(Group)
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)
	Users.UpsertUser(user)

	_, err := Users.GetUserByNonce(user.Nonce)
	if err != nil {
		t.Errorf("GetUserByNonce: Expected to find user by nonce!")
	}
}

// Make sure the nonce converts correctly to and from storage
func TestUserNonceConversion(t *testing.T) {
	InitCrypto()

	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.User]*User),
		collectionLock: &sync.Mutex{},
	})

	user := Users.NewUser(Group)
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)
	Users.UpsertUser(user)

	testUser, _ := Users.GetUserByNonce(user.Nonce)
	if bytes.Equal(testUser.Nonce.Bytes(), user.Nonce.Bytes()) {
		t.Errorf("UserNonceConversion: Expected nonces to match! %v %v",
			Group.NewIntFromBytes(testUser.Nonce.Bytes()),
			Group.NewIntFromBytes(user.Nonce.Bytes()))
	}
	if !testUser.Nonce.GenTime.Equal(user.Nonce.GenTime) {
		t.Errorf("UserNonceConversion: Expected GenTime to match! %v %v",
			testUser.Nonce.GenTime, user.Nonce.GenTime)
	}
	if testUser.Nonce.TTL != user.Nonce.TTL {
		t.Errorf("UserNonceConversion: Expected TTL to match! %v %v",
			testUser.Nonce.TTL, user.Nonce.TTL)
	}
	if !testUser.Nonce.ExpiryTime.Equal(user.Nonce.ExpiryTime) {
		t.Errorf("UserNonceConversion: Expected ExpiryTime to match! %v %v",
			testUser.Nonce.ExpiryTime, user.Nonce.ExpiryTime)
	}
}
