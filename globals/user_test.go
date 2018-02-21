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

	if Users.CountUsers() != 0 {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}

	Users.UpsertUser(testUser)

	if Users.CountUsers() != 1 {
		t.Errorf("UpsertUser: Failed to add a new user!")
	}

	if getUser, _ := Users.GetUser(testUser.Id); getUser != testUser {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	Users.DeleteUser(testUser.Id)

	if _, userExists := Users.GetUser(testUser.Id); userExists {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}

	if Users.CountUsers() != 0 {
		t.Errorf("DeleteUser: Excepted empty userRegistry after user deletion!")
	}

}
