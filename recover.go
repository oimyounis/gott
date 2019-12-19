package gott

import (
	"fmt"
	"log"
	"runtime/debug"
)

func Recover() {
	if r := recover(); r != nil {
		log.Println("Recovered from panic:", r)
		stack := string(debug.Stack())
		fmt.Println(stack)
		// TODO: report panic to remote service and log to file
	}
}
