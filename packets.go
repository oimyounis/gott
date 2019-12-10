package gott

type ConnectFlags struct {
	Reserved, CleanSession, WillFlag, WillQOS, WillRetain, PasswordFlag, UserNameFlag string
}

func CheckConnectFlags(bits string) ConnectFlags {
	return ConnectFlags{
		Reserved:     bits[7:],
		CleanSession: bits[6:7],
		WillFlag:     bits[5:6],
		WillQOS:      bits[3:5],
		WillRetain:   bits[2:3],
		PasswordFlag: bits[1:2],
		UserNameFlag: bits[0:1],
	}
}

func GetFixedHeader(packetType byte) []byte {
	h := make([]byte, FIXED_HEADER_LEN)
	h[0] = packetType

	if packetType == TYPE_CONNACK_BYTE {
		h[1] = 2
	}
	return h
}

func MakeConnAckPacket(sessionPresent, returnCode byte) (packet []byte) {
	packet = append(packet, GetFixedHeader(TYPE_CONNACK_BYTE)...)
	packet = append(packet, sessionPresent, returnCode)
	return
}
