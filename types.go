package gott

import "time"

type ConnectFlags struct {
	CleanSession, WillFlag, WillRetain, PasswordFlag, UserNameFlag bool
	WillQoS                                                        byte
}

type PublishFlags struct {
	DUP, QoS byte
	Retain   bool
	PacketId uint16
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
	Retain         bool
	Timestamp      time.Time // used for sorting retained messages in the order they were received (less is first)
}

type PacketStatus struct {
	QoS, Status byte
}
