package server

import (
	"reflect"
	"testing"
)

// tests that the proper queue is returned
func TestFirstNode_GetNewBatchQueue(t *testing.T) {
	fn := &firstNode{}
	fn.Initialize()

	if !reflect.DeepEqual(fn.newBatchQueue, fn.GetNewBatchQueue()) {
		t.Errorf("FirstNode.GetNewBatchQueue: returned queue not the same" +
			" as internal queue")
	}
}

// tests that the proper queue is returned
func TestFirstNode_GetCompletedPrecompQueue(t *testing.T) {
	fn := &firstNode{}
	fn.Initialize()

	if !reflect.DeepEqual(fn.readyRounds, fn.GetCompletedPrecomps()) {
		t.Errorf("FirstNode.GetCompletedPrecompQueue: returned queue not the same" +
			" as internal queue")
	}
}
