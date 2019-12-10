package main

import (
	"gott"
)

func main() {
	broker := gott.NewBroker(":1883")
	if err := broker.Listen(); err != nil {
		panic(err)
	}
}
