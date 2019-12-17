package gott

import "log"

func Recover() {
	if r := recover(); r != nil {
		log.Println("Recovered from panic:", r)
		// TODO: report panic to remote service
	}
}
