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
	Users = NewUserRegistry("cmix", "",
		"cmix_server", "")

	// Loop from userDatabase.go to create and add users
	nickList := []string{"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
		"Mario", "Will", "Sydney", "Jon0"}

	for i := 1; i <= len(nickList); i++ {
		u := Users.NewUser("")
		u.Nick = nickList[i-1]
		Users.UpsertUser(u)
	}
	for i := len(nickList) + 1; i <= NUM_DEMO_USERS; i++ {
		u := Users.NewUser("")
		u.Nick = ""
		Users.UpsertUser(u)
	}

	// TESTS Start here
	test := 7
	pass := 0

	numUsers := Users.CountUsers()

	if numUsers != NUM_DEMO_USERS {
		t.Errorf("Count users is not working correctly. "+
			"Expected: %v Actual: %v", NUM_DEMO_USERS, numUsers)
	} else {
		pass++
	}

	usr9, _ := Users.GetUser(9)

	if usr9 == nil {
		t.Errorf("Error fetching user!")
	} else {
		pass++
	}

	getUser, err := Users.GetUser(usr9.ID)

	if (err != nil) || getUser.ID != usr9.ID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	usr3, _ := Users.GetUser(3)
	usr5, _ := Users.GetUser(5)

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

	ids, _ := Users.GetNickList()

	if len(ids) != Users.CountUsers() {
		t.Errorf("Nicklist is not ok! ")
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
