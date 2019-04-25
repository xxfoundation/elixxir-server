////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/server"
)

func NewServerImplementation(instance *server.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Phase = func(batch *mixmessages.CmixBatch) { ReceivePhase(instance, batch) }
	return impl
}
