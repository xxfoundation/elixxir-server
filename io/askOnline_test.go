package io

import (
	"net"
	"testing"
)

func TestVerifyServersOnline(t *testing.T) {

	// (Static) IP of the last node
	server := "13.56.70.255:11420"

	test := 1
	pass := 0

	conn, err := net.Dial("tcp", server)
	if err != nil {
		t.Errorf("Error:Program could not connect to last node!")
	} else {
		pass++
	}

	defer conn.Close()

	println("AskOnline test", pass, "out of", test, "tests passed.")
}
