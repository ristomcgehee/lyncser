package utils

import (
	"os"
	"strings"

	"mvdan.cc/sh/v3/shell"
)

const (
	TimeFormat = "2006-01-02T15:04:05.000Z"
)

func RealPath(path string) string {
	escapedPath := strings.ReplaceAll(path, "'", "\\'")
	out, err := shell.Fields(escapedPath, nil)
	PanicError(err)
	return out[0]
}

// Panics if the error is not nil.
func PanicError(err error) {
	if err != nil {
		panic(err)
	}
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	PanicError(err)
	return true
}
