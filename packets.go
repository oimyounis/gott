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

type PublishFlags struct {
	DUP, QoS, Retain string
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

func ExtractPublishFlags(bits string) PublishFlags {
	return PublishFlags{
		DUP:    bits[0:1],
		QoS:    bits[1:3],
		Retain: bits[3:],
	}
}

func ParseFixedHeaderFirstByte(b byte) (byte, string) {
	bs := bytes.ByteToBinaryString(b)
	packetType, err := bytes.BinaryStringToByte(bs[:4])
	if err != nil {
		return 0, ""
	}
	return packetType, bs[4:]
}

func MakeConnAckPacket(sessionPresent, returnCode byte) []byte {
	return []byte{TYPE_CONNACK_BYTE, CONNECT_REM_LEN, sessionPresent, returnCode}
}

// Not completed yet
func MakePublishPacket(topic, payload []byte, dupFlag, qos, retainFlag byte) (packet []byte) {
	// TODO: complete the implementation of this func
	if topic == nil {
		return nil
	}

	// 2nd byte = remaining length (var header len + payload len)
	fixedHeader := []byte{TYPE_PUBLISH_BYTE + dupFlag<<3 + qos<<1 + retainFlag, 0}

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
