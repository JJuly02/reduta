package store

import "encoding/hex"

func hexDecode(s string) ([]byte, error) { return hex.DecodeString(s) }
