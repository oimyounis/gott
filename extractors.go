package gott

import (
	"encoding/binary"
	"errors"
	"gott/bytes"
)

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

func ExtractSubTopicFilters(payload []byte) ([]Filter, error) {
	filterList := make([]Filter, 0)
	parsedBytes := 0
	payloadLen := len(payload)

	for parsedBytes < payloadLen {
		lenBytesEnd := parsedBytes + 2
		topicLen := int(binary.BigEndian.Uint16(payload[parsedBytes:lenBytesEnd]))
		topicNameEnd := lenBytesEnd + topicLen
		topicFilter := string(payload[lenBytesEnd:topicNameEnd])
		qos := payload[topicNameEnd : topicNameEnd+1][0]

		// TODO: add qos validation

		filterList = append(filterList, Filter{topicFilter, qos})

		parsedBytes += topicLen + 3 // 3 is 2 bytes for topic length and 1 for qos
	}

	if parsedBytes != payloadLen {
		return nil, errors.New("invalid payload")
	}

	return filterList, nil
}

func ParseFixedHeaderFirstByte(b byte) (byte, string) {
	bs := bytes.ByteToBinaryString(b)
	packetType, err := bytes.BinaryStringToByte(bs[:4])
	if err != nil {
		return 0, ""
	}
	return packetType, bs[4:] // packet type and flags bits (eg. 1101)
}
