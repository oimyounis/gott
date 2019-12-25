package gott

import "time"

type connectFlags struct {
	CleanSession, WillFlag, WillRetain, PasswordFlag, UserNameFlag bool
	WillQoS                                                        byte
}

type publishFlags struct {
	DUP, QoS byte
	Retain   bool
	PacketID uint16
}

type filter struct {
	Filter []byte
	QoS    byte
}

type subscription struct {
	Session *session
	QoS     byte
}

type message struct {
	Topic, Payload []byte
	QoS            byte
	Retain         bool
	Timestamp      time.Time // used for sorting retained messages in the order they were received (less is first)
}
