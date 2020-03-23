// Package common contains various helper functions.
package common

import (
	"encoding/binary"
	"encoding/hex"
)

// FromHex convert hex byte string to []byte
func FromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" {
			s = s[2:]
		}
		if len(s)%2 == 1 {
			s = "0" + s
		}
		return Hex2Bytes(s)
	}
	return nil
}

// Bytes2Hex convert byte array to string
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// Hex2Bytes convert hex string to byte array
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// Unit64ToBytes convert uint64 to bytes
func Unit64ToBytes(n uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	return buf
}

// BytesToUnit64 convert bytes to uint64
func BytesToUnit64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
