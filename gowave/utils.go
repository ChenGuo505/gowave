package gowave

import "strings"

func TrimPrefix(str string, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str[len(prefix):]
	}
	return str
}
