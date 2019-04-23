package server

import (
	"encoding/binary"
	"gitlab.com/elixxir/server/node"
)

type PhaseFingerprint [9]byte

func makeGraphFingerprint(rid node.RoundID, p node.PhaseType) PhaseFingerprint {
	var gf PhaseFingerprint
	binary.BigEndian.PutUint64(gf[:8], uint64(rid))
	gf[8] = byte(p)
	return gf
}
