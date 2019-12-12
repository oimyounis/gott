package gott

type ConnectFlags struct {
	Reserved, CleanSession, WillFlag, WillQOS, WillRetain, PasswordFlag, UserNameFlag string
}

type PublishFlags struct {
	DUP, QoS, Retain string
}

type Filter struct {
	Filter string
	QoS    byte
}
