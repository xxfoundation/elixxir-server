package node

import (
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the userRegistry map
func TestUserRegistry(t *testing.T) {

	userRegistry = make(map[uint64]User)
	testUser := NewUser(uint64(1), "Someplace")

	if CountUsers() != 0 {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}

	UpsertUser(testUser)

	if CountUsers() != 1 {
		t.Errorf("UpsertUser: Failed to add a new user!")
	}

	if getUser := GetUser(testUser.Id); getUser != testUser {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	DeleteUser(testUser.Id)

	if getUser := GetUser(testUser.Id); getUser.Address != "" {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}

	if CountUsers() != 0 {
		t.Errorf("DeleteUser: Excepted empty userRegistry after user deletion!")
	}

}
