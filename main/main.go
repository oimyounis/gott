package main

import (
	"gott"
	"gott/bytes"
	"log"
)

func main() {
	log.Println(bytes.BinaryStringToByte("0011"))
	//gott.MakePublishPacket([]byte("aa/b"), []byte(""), 1, 1, 1)
	//gott.MakePublishPacket([]byte("aa/b"), []byte(""), 1, 0, 1)
	//gott.MakePublishPacket([]byte("a/b"), []byte(""), 0, 2, 0)
	broker := gott.NewBroker(":1883")
	if err := broker.Listen(); err != nil {
		panic(err)
	}
}
