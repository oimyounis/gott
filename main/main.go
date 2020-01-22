package main

import (
	"gott"
	"gott/dashboard"
	"log"
)

func main() {
	broker, err := gott.NewBroker()
	if err != nil {
		panic(err)
	}

	go dashboard.Serve(broker, ":18830")

	if err = broker.Listen(); err != nil {
		dashboard.Stop()
		log.Fatalln(err)
	}
}
