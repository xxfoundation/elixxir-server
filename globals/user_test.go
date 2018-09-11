////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"sync"
	"testing"
	"gitlab.com/privategrity/crypto/id"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
// TODO: This test needs split up
func TestUserRegistry(t *testing.T) {
	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.UserID]*User),
		collectionLock: &sync.Mutex{},
	})

	for i := 0; i < NUM_DEMO_USERS; i++ {
		u := Users.NewUser("")
		u.Nick = ""
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

	usr9, _ := Users.GetUser(id.NewUserIDFromUint(9, t))

	if usr9 == nil {
		t.Errorf("Error fetching user!")
	} else {
		pass++
	}

	getUser, err := Users.GetUser(usr9.ID)

	if (err != nil) || getUser.ID != usr9.ID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	usr3, _ := Users.GetUser(id.NewUserIDFromUint(3, t))
	usr5, _ := Users.GetUser(id.NewUserIDFromUint(5, t))

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
	fk := ForwardKey{
		BaseKey:      cyclic.NewInt(10),
		RecursiveKey: cyclic.NewInt(15),
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
	Users := UserRegistry(&UserMap{
		userCollection: make(map[id.UserID]*User),
		collectionLock: &sync.Mutex{},
	})
	user := Users.NewUser("t")
	user.Transmission.BaseKey = cyclic.NewInt(66)

	newUser := user.DeepCopy()
	if user.Transmission.BaseKey.Cmp(newUser.Transmission.BaseKey) != 0 {
		t.Errorf("User Deepcopy: Failed to copy keys!")
	}
}
