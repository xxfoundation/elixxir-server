////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserRegistry(t *testing.T) {

	testUser := Users.NewUser("Someplace")
	testUser.Nick = "Me"
	// TODO see tests at bottom of file. removed numUsers temporarily
	//numUsers := Users.CountUsers()
	Users.DeleteUser(testUser.UID)
	Users.UpsertUser(testUser)
	getUser, exists := Users.GetUser(testUser.UID)

	if !exists || getUser.UID != testUser.UID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	getUser.Transmission.RecursiveKey.SetInt64(5)
	getUser.Nick = "Michael"

	Users.UpsertUser(getUser)

	getUser2, _ := Users.GetUser(testUser.UID)

	if getUser2.Transmission.RecursiveKey.Int64() != 5 || getUser2.
		Nick != "Michael" {
		t.Errorf("UpsertUser: User did not save! Got: %v, %v; expected: %v, " +
			"%v", getUser2.Transmission.RecursiveKey.Int64(), getUser2.Nick,
				5, "Michael")
	}

	Users.DeleteUser(testUser.UID)

	// TODO Fix these tests to work with the hard-coded users
/*
	if _, userExists := Users.GetUser(testUser.UID); userExists {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}

	if count := Users.CountUsers(); count != numUsers {
		t.Errorf("DeleteUser: Excepted empty userRegistry after user"+
			" deletion! Got %d expected %d", count, numUsers)
	}
*/
}
