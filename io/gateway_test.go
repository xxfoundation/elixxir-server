////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
)

func TestGetRoundBufferInfo(t *testing.T) {
	GetRoundBufferInfoTimeout = "1s"
	l1, e1 := GetRoundBufferInfo()
	if l1 != 0 || e1 == nil {
		t.Errorf("GetRoundBufferInfo should return 0 with err, instead got: %v, %v",
			l1, e1)
	}
	RoundCh = make(chan *string, 10)
	RoundCh <- &GetRoundBufferInfoTimeout // Note: any string would do...
	l2, e2 := GetRoundBufferInfo()
	if l2 != 1 || e2 != nil {
		t.Errorf("GetRoundBufferInfo should return 1, nil, instead got: %v, %v",
			l2, e2)
	}
}
