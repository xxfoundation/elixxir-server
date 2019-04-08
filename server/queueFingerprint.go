package server

import (
	"encoding/binary"
	"gitlab.com/elixxir/server/node"
)

type QueueFingerprint [9]byte

func makeGraphFingerprint(rid node.RoundID, p node.Phase) QueueFingerprint {
	var gf QueueFingerprint
	binary.BigEndian.PutUint64(gf[:8], uint64(rid))
	gf[8] = byte(p)
	return gf
}
