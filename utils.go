package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/alessio/shellescape"
)

func realPath(path string) string {
	out, err := exec.Command("bash", "-c", "readlink -m "+shellescape.StripUnsafe(path)).Output()
	checkError(err)
	return strings.TrimSpace(string(out[:]))
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	checkError(err)
	return true
}
