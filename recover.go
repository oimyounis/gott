package gott

import "log"

func Recover() interface{} {
	r := recover()
	if r != nil {
		log.Println("Recovered from panic:", r)
		// TODO: report panic to remote service
	}
	return r
}
