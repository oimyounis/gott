package gott

type ConnectFlags struct {
	Reserved, CleanSession, WillFlag, WillQOS, WillRetain, PasswordFlag, UserNameFlag string
}

type PublishFlags struct {
	DUP, QoS, Retain byte
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
}

type PacketStatus struct {
	QoS, Status byte
}
