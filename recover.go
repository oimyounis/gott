package gott

import (
	"log"
)

func Recover() {
	if r := recover(); r != nil {
		log.Println("Recovered from panic:", r)
		//stack := string(debug.Stack())
		// TODO: report panic to remote service and log to file
	}
}
