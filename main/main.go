package main

import (
	"gott"
	"log"
)

func main() {
	broker := gott.NewBroker(":1883")
	if err := broker.Listen(); err != nil {
		log.Fatalln(err)
	}
}
