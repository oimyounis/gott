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
