//go:build !appengine
// +build !appengine

package fasttemplate

import (
	"unsafe"
)

func unsafeBytes2String(b []byte) string {
	// Copied from https://go101.org/article/unsafe.html
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func unsafeString2Bytes(s string) []byte {
	// Copied from https://go101.org/article/unsafe.html
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
