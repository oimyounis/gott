package main

import (
	"net"
)

// func Bootstrap() {}
// func Bootstrap(conf map[interface{}]interface{}) {}

func OnSocketOpen(conn net.Conn) bool {
	return true
}

func OnBeforeConnect(clientID, username, password string) bool {
	return true
}

func OnConnect(clientID, username, password string) bool {
	return true
}

func OnMessage(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) {}

func OnBeforePublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool {
	return true
}

func OnPublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) {}

func OnBeforeSubscribe(clientID, username string, topic []byte, qos byte) bool {
	return true
}

func OnSubscribe(clientID, username string, topic []byte, qos byte) {}

func OnBeforeUnsubscribe(clientID, username string, topic []byte) bool {
	return true
}

func OnUnsubscribe(clientID, username string, topic []byte) {}

func OnDisconnect(clientID, username string, graceful bool) {}

func Cleanup() {}
