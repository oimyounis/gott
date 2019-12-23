package gott

import (
	"log"
	"os"
)

var Log = func(prefix string, v ...interface{}) {
	l := log.New(os.Stdout, prefix+" ", log.LstdFlags)
	l.Println(v...)
}
