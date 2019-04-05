package server

import (
	"encoding/binary"
	"gitlab.com/elixxir/server/globals"
)

type GraphFingerprint [9]byte

func makeGraphFingerprint(rid globals.RoundID, p globals.Phase) GraphFingerprint {
	var gf GraphFingerprint
	binary.BigEndian.PutUint64(gf[:8], uint64(rid))
	gf[8] = byte(p)
	return gf
}
