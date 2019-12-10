package gott

func GetFixedHeader(packetType int) [2]byte {
	h := [2]byte{}
	h[0] = byte(packetType << 4)
	return h
}
