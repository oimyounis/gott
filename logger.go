package gott

import (
	"log"
)

// LogBench logs to stdout with a timestamp and a "[BENCHMARK]" prefix.
func LogBench(v ...interface{}) {
	printList := []interface{}{"[BENCHMARK]"}
	printList = append(printList, v...)
	log.Println(printList...)
}

func LogDebug(v ...interface{}) {
	printList := []interface{}{"[DEBUG]"}
	printList = append(printList, v...)
	log.Println(printList...)
}
