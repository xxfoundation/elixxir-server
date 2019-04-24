package server

import "gitlab.com/elixxir/server/node"

type PhaseFingerprint struct {
	phase node.PhaseType
	round node.RoundID
}
