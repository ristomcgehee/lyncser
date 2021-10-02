package main

import (
	"os"
	
	"mvdan.cc/sh/v3/shell"
)

func realPath(path string) string {
	out, err := shell.Fields(path, nil)
	checkError(err)
	return out[0]
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
