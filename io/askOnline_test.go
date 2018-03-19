package io

import (
	"gitlab.com/privategrity/comms/clusterclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"testing"
	"time"
)

func TestVerifyServersOnline(t *testing.T) {

	// This is the (static) IP of the last node
	servers := []string{"13.56.70.255:11420"}

	test := len(servers) + 1
	pass := 0

	for i := 0; i < len(servers); {
		_, err := clusterclient.SendAskOnline(servers[i], &pb.Ping{})

		if err != nil {
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
			pass++
		}
	}

	fakeServer := "SuperMario"
	_, e := clusterclient.SendAskOnline(fakeServer, &pb.Ping{})

	if e != nil {
		pass++
	} else {
		t.Errorf("Error:Program connnected to a fake server!")
	}

	println("AskOnline test", pass, "out of", test, "tests passed.")
}
