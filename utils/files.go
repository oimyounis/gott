package utils

import (
	"io/ioutil"
	"log"
	"os"
)

func ReadFile(path string) string {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("Error reading file: "+path, err)
		return ""
	}
	return string(file)
}

func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
