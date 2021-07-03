package main

import (
	"gott"
	"log"
)

func main() {
	broker, err := gott.NewBroker()
	if err != nil {
		panic(err)
	}

	if err = broker.Listen(); err != nil {
		log.Fatalln(err)
	}
}
