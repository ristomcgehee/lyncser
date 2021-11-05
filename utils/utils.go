package utils

import (
	"os"
	"strings"
	"time"

	"mvdan.cc/sh/v3/shell"
)

const (
	TimeFormat = "2006-01-02T15:04:05.000Z"
	// Used to indicate a file has never been synced.
	neverSyncedStr = "2000-01-01T01:01:01.000Z"
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

// Remove returns a slice with the first item that satisfies f removed. Order is retained. This can be an expensive
// operation if there are many items in slice.
func Remove(f func(item string) bool, slice []string) []string {
	for i, item := range slice {
		if f(item) {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// InSlice returns true if item is present in slice.
func InSlice(item string, slice []string) bool {
	for _, sliceItem := range slice {
		if item == sliceItem {
			return true
		}
	}
	return false
}

// Returns true if the file has been synced based on the last cloud update.
func HasBeenSynced(lastCloudUpdate time.Time) bool {
	neverSynced, _ := time.Parse(TimeFormat, neverSyncedStr)
	return lastCloudUpdate.After(neverSynced)
}

// Returns a time used to signify that a file has never been synced.
func GetNeverSynced() time.Time {
	neverSynced, _ := time.Parse(TimeFormat, neverSyncedStr)
	return neverSynced
}
