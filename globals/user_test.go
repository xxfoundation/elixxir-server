////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

const numTestDemoUsers = 256

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
// TODO: This test needs split up
func TestUserRegistry(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})

	for i := 0; i < numTestDemoUsers; i++ {
		u := users.NewUser(grp)
		users.UpsertUser(u)
	}

	// TESTS Start here

	numUsers := users.CountUsers()

	if numUsers != numTestDemoUsers {
		t.Errorf("Count users is not working correctly. "+
			"Expected: %v Actual: %v", numTestDemoUsers, numUsers)
	}

	id9 := id.NewIdFromUInt(9, id.User, t)
	usr9, err := users.GetUser(id9)

	if err != nil {
		t.Errorf("User fetch returned error: %s", err.Error())
	}

	if usr9 == nil {
		t.Fatalf("Error fetching user!")
	}

	getUser, err := users.GetUser(usr9.ID)

	if (err != nil) || getUser.ID != usr9.ID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	usr3, _ := users.GetUser(id.NewIdFromUInt(3, id.User, t))
	usr5, _ := users.GetUser(id.NewIdFromUInt(5, id.User, t))

	if usr3.BaseKey == usr5.BaseKey {
		t.Errorf("Transmissions keys are the same and they should be different!")
	}

	users.DeleteUser(usr9.ID)

	if users.CountUsers() != numTestDemoUsers-1 {
		t.Errorf("User has not been deleted correctly. "+
			"Expected # of users: %v Actual: %v", numTestDemoUsers-1, users.CountUsers())
	}

	if _, userExists := users.GetUser(usr9.ID); userExists == nil {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}
}

// Test happy path
func TestUser_DeepCopy(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})

	user := users.NewUser(grp)
	user.BaseKey = grp.NewInt(66)

	newUser := user.DeepCopy()
	if user.BaseKey.Cmp(newUser.BaseKey) != 0 {
		t.Errorf("User Deepcopy: Failed to copy keys!")
	}

	var uNil *User

	uNilCpy := uNil.DeepCopy()

	if uNilCpy != nil {
		t.Errorf("User Deepcopy: copy occured on nil user")
	}
}

// Test happy path and inserting too many salts
func TestUserMap_InsertSalt(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})
	u9 := users.NewUser(grp)
	u9.ID = id.NewIdFromUInt(1, id.User, t)
	users.UpsertUser(u9)

	// Insert like 300 salts, expect success
	for i := 0; i < MaxSalts; i++ {
		err := users.InsertSalt(u9.ID, []byte("test"))
		if err != nil {
			t.Errorf("InsertSalt: Expected success! Recieved: %s", err.Error())
		}
	}
	// Now we have exceeded the max number, expect failure
	err := users.InsertSalt(u9.ID, []byte("test"))
	if err == nil {
		t.Errorf("InsertSalt: Expected failure due to exceeding max count of" +
			" salts for one user, recieved success")
	}
}
