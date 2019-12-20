package gott

import (
	"fmt"
	"log"
	"runtime/debug"
)

func Recover(callback func(err, stack string)) {
	if r := recover(); r != nil {
		log.Println("Recovered from panic:", r)
		stack := string(debug.Stack())
		fmt.Println(stack)
		// TODO: report panic to remote service and log to file

		if callback != nil {
			callback(fmt.Sprintf("%s", r), stack)
		}
	}
}
