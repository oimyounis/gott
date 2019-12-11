package gott

import (
	"encoding/binary"
	"gott/bytes"
	"log"
)

var packetSeq *Sequencer = &Sequencer{UpperBoundBits: 16, Start: 1}

type ConnectFlags struct {
	Reserved, CleanSession, WillFlag, WillQOS, WillRetain, PasswordFlag, UserNameFlag string
}

func ExtractConnectFlags(bits string) ConnectFlags {
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

func GetPacketType(b byte) byte {
	bs := bytes.ByteToBinaryString(b)
	packetTypeHalf := bs[:4]
	packetType, err := bytes.BinaryStringToByte(packetTypeHalf)
	if err != nil {
		return 0
	}
	return packetType
}

func GetFixedHeader(packetType byte) []byte {
	h := make([]byte, FIXED_HEADER_LEN)
	h[0] = packetType

	if packetType == TYPE_CONNACK_BYTE {
		h[1] = 2 // constant remaining len as per [3.2.1]
	}
	return h
}

func MakeConnAckPacket(sessionPresent, returnCode byte) (packet []byte) {
	packet = append(packet, GetFixedHeader(TYPE_CONNACK_BYTE)...)
	packet = append(packet, sessionPresent, returnCode) // variable header as per [3.2.2]
	return
}

func MakePublishPacket(topic, payload []byte, dupFlag, qos, retainFlag byte) (packet []byte) {
	if topic == nil {
		return nil
	}

	fixedHeader := make([]byte, FIXED_HEADER_LEN)
	byte1 := TYPE_PUBLISH_BYTE + dupFlag<<3 + qos<<1 + retainFlag

	fixedHeader[0] = byte1
	fixedHeader[1] = 0 // remaining length (var header len + payload len)

	varHeader := make([]byte, 2) // topic name + packet identifier
	binary.BigEndian.PutUint16(varHeader, uint16(len(topic)))
	varHeader = append(varHeader, topic...)

	if qos >= 1 {
		varHeader = append(varHeader, 0, 0)
		binary.BigEndian.PutUint16(varHeader[len(varHeader)-2:], uint16(packetSeq.Next()))
	}

	log.Println(fixedHeader, "-", varHeader)
	return
}
