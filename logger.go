package gott

import (
	"log"
)

var LogBench = func(v ...interface{}) {
	printList := []interface{}{"[BENCHMARK]"}
	printList = append(printList, v...)
	log.Println(printList...)
}
