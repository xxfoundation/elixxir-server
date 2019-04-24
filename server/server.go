package server

import "gitlab.com/elixxir/crypto/cyclic"

type ServerContext struct {
	roundManager  *RoundManager
	resourceQueue *ResourceQueue
	grp           *cyclic.Group
}

//fixme: move newound to run off server
//fixme: write initlizer, including queue
