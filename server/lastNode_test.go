package server

import (
	"reflect"
	"testing"
)

// tests that the proper queue is returned
func TestLastNode_GetCompletedBatchQueue(t *testing.T) {
	ln := &LastNode{}
	ln.Initialize()

	if !reflect.DeepEqual(ln.completedBatchQueue, ln.GetCompletedBatchQueue()) {
		t.Errorf("LastNode.GetCompletedBatchQueue: returned queue not the same" +
			" as internal queue")
	}
}
