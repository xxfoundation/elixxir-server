package globals

import (
	"testing"
	//"gitlab.com/privategrity/crypto/cyclic"
)

func TestTestDB(t *testing.T) {
	database := InitDatabase()

	//testUser := &User{Id: uint64(5), Address: "TEsfasfsST",
	//	Transmission: ForwardKey{BaseKey: cyclic.NewInt(444444),
	//		RecursiveKey: cyclic.NewInt(4444)},
	//	Reception: ForwardKey{BaseKey: cyclic.NewInt(444444),
	//		RecursiveKey: cyclic.NewInt(333344433)},
	//	PublicKey: cyclic.NewInt(5555444456)}

	database.DeleteUser(uint64(55))
	user, ok := database.GetUser(uint64(1))
	println(user)
	println(ok)
}
