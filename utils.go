package main

import (
	"os/exec"
	"strings"

	"github.com/alessio/shellescape"
)

func realPath(path string) string {
	out, err := exec.Command("bash", "-c", "realpath "+shellescape.StripUnsafe(path)).Output()
	checkError(err)
	return strings.TrimSpace(string(out[:]))
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
