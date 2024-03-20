package zteapi

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func EncodePass(pass, ld string) string {
	pass = strings.ToUpper(pass)
	b := sha256.Sum256([]byte(pass + ld))
	return strings.ToUpper(hex.EncodeToString(b[:]))
}

func EncodeAD(version, rd string) string {
	r0r1 := EncodePass(version, "")
	return EncodePass(r0r1, rd)
}
