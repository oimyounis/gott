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
	Filter []byte `json:"filter"`
	QoS    byte   `json:"qos"`
}

type subscription struct {
	Session *session `json:"session"`
	QoS     byte     `json:"qos"`
}

type message struct {
	Topic, Payload []byte
	QoS            byte
	Retain         bool
	Timestamp      time.Time // used for sorting retained messages in the order they were received (less is first)
}
