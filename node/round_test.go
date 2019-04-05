package node

import (
	"testing"
	"math/rand"
)

func TestNewRound(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests:= 30

	for i:=0;i<tests;i++{
		batchSize := rng.Uint32()%1000
		expandedBatchSize := batchSize*2


	}


}