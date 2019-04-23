////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package server

import (
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"sync"
)

type queueElement struct {
	g     *services.Graph
	id    node.RoundID
	phase node.Phase
	loc   int
	sync.Mutex
}

func (qe *queueElement) GetFingerprint() QueueFingerprint {
	return makeGraphFingerprint(qe.id, qe.phase)
}
