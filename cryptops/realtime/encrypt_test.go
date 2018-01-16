package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

// For now this unit test only tests for regression, not correctness
func TestRealtimeEncrypt(t *testing.T) {
	test := 2
	pass := 0
	batchSize := uint64(2)
	round := server.NewRound(batchSize)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	group := cyclic.NewGroup(cyclic.NewInt(21),
		cyclic.NewInt(17), cyclic.NewInt(23), rng)

	round.Z = cyclic.NewInt(9)
	round.T[0] = cyclic.NewInt(17)
	round.T[1] = cyclic.NewInt(14)

	var inMessages []*services.Message
	inMessages = append(inMessages, &services.Message{
		uint64(0), []*cyclic.Int{cyclic.NewInt(6)}})
	inMessages = append(inMessages, &services.Message{
		uint64(0), []*cyclic.Int{cyclic.NewInt(18)}})

	dispatch := services.DispatchCryptop(&group, RealtimeEncrypt{},
		nil, nil, round)

	expected := []*cyclic.Int{cyclic.NewInt(15), cyclic.NewInt(3)}

	for i := 0; i < len(inMessages); i++ {
		dispatch.InChannel <- inMessages[i]
		actual := <-dispatch.OutChannel
		if expected[i].Cmp(actual.Data[0]) == 0 {
			pass++
		} else {
			t.Error("Test failed at index", i)
			t.Error("Actual:", actual.Data[0].Text(10), "expected:", expected[i].Text(10))
		}
	}

	println("RealtimeEncrypt:", pass, "out of", test, "passed")
}
