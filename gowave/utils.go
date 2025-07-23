package gowave

import (
	"strings"
	"unicode"
	"unsafe"
)

func TrimPrefix(str string, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str[len(prefix):]
	}
	return str
}

func IsASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)}))
}
