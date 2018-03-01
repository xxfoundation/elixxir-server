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
	numUsers := Users.CountUsers()
	Users.DeleteUser(testUser.Id)
	Users.UpsertUser(testUser)
	getUser, exists := Users.GetUser(testUser.Id)

	if !exists || getUser.Id != testUser.Id {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	getUser.Transmission.RecursiveKey.SetInt64(5)

	Users.UpsertUser(getUser)

	getUser2, _ := Users.GetUser(testUser.Id)

	if getUser2.Transmission.RecursiveKey.Int64() != 5 {
		t.Errorf("UpsertUser: User did not save!")
	}

	Users.DeleteUser(testUser.Id)

	if _, userExists := Users.GetUser(testUser.Id); userExists {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}

	if count := Users.CountUsers(); count != numUsers {
		t.Errorf("DeleteUser: Excepted empty userRegistry after user"+
			" deletion! Got %d expected %d", count, numUsers)
	}

}
