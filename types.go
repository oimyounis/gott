package gott

import "time"

type ConnectFlags struct {
	Reserved, CleanSession, WillFlag, WillQOS, WillRetain, PasswordFlag, UserNameFlag string
}

type PublishFlags struct {
	DUP, QoS, Retain byte
	PacketId         uint16
}

type Filter struct {
	Filter []byte
	QoS    byte
}

type Subscription struct {
	Client *Client
	QoS    byte
}

type Message struct {
	Topic, Payload []byte
	QoS            byte
	Timestamp      time.Time // used for sorting retained messages in the order they where received (less is first)
}

type PacketStatus struct {
	QoS, Status byte
}
