package gott

import (
	"encoding/binary"
	"errors"

	"github.com/oimyounis/gott/bytes"
)

var noopConnectFlags = connectFlags{}

func extractConnectFlags(b byte) (connectFlags, error) {
	var willQos byte
	qosBit3 := bytes.BitIsSet(b, 3)
	qosBit4 := bytes.BitIsSet(b, 4)
	willFlag := bytes.BitIsSet(b, 2)
	willRetain := bytes.BitIsSet(b, 5)
	reservedBit := bytes.BitIsSet(b, 0)

	if reservedBit { // as per [MQTT-3.1.2-3]
		return noopConnectFlags, errors.New("reserved bit is set in connect flag bits")
	}

	if qosBit3 && qosBit4 {
		return noopConnectFlags, errors.New("invalid QoS in connect flag bits")
	} else if qosBit3 {
		willQos = 1
	} else if qosBit4 {
		willQos = 2
	}

	if !willFlag && willQos != 0 {
		return noopConnectFlags, errors.New("will flag is set to 0 and will QoS is not 0")
	}
	if !willFlag && willRetain {
		return noopConnectFlags, errors.New("will flag is set to 0 and will retain is not 0")
	}
	return connectFlags{
		CleanSession: bytes.BitIsSet(b, 1),
		WillFlag:     willFlag,
		WillQoS:      willQos,
		WillRetain:   willRetain,
		PasswordFlag: bytes.BitIsSet(b, 6),
		UserNameFlag: bytes.BitIsSet(b, 7),
	}, nil
}

var noopPublishFlags = publishFlags{}

func extractPublishFlags(bits string) (publishFlags, error) {
	var (
		qos, dup byte
		retain   bool
	)

	switch bits[1:3] {
	case "00":
		qos = 0
	case "01":
		qos = 1
	case "10":
		qos = 2
	default:
		return noopPublishFlags, errors.New("invalid flags in publish packet")
	}

	switch bits[3:] {
	case "0":
		retain = false
	case "1":
		retain = true
	default:
		return noopPublishFlags, errors.New("invalid flags in publish packet")
	}

	switch bits[0:1] {
	case "0":
		dup = 0
	case "1":
		dup = 1
	default:
		return noopPublishFlags, errors.New("invalid flags in publish packet")
	}

	return publishFlags{
		DUP:    dup,
		QoS:    qos,
		Retain: retain,
	}, nil
}

func extractSubTopicFilters(payload []byte) ([]filter, error) {
	filterList := make([]filter, 0)
	parsedBytes := 0
	payloadLen := len(payload)

	for parsedBytes < payloadLen {
		lenBytesEnd := parsedBytes + 2
		topicLen := int(binary.BigEndian.Uint16(payload[parsedBytes:lenBytesEnd]))
		topicNameEnd := lenBytesEnd + topicLen
		topicFilter := payload[lenBytesEnd:topicNameEnd]
		qos := payload[topicNameEnd : topicNameEnd+1][0]

		if qos < 0 && qos > 2 {
			return nil, errors.New("invalid QoS")
		}

		filterList = append(filterList, filter{topicFilter, qos})

		parsedBytes += topicLen + 3 // 3 is 2 bytes for topic length and 1 for qos
	}

	if parsedBytes != payloadLen {
		return nil, errors.New("invalid payload")
	}

	return filterList, nil
}

func extractUnSubTopicFilters(payload []byte) ([][]byte, error) {
	filterList := make([][]byte, 0)
	parsedBytes := 0
	payloadLen := len(payload)

	for parsedBytes < payloadLen {
		lenBytesEnd := parsedBytes + 2
		topicLen := int(binary.BigEndian.Uint16(payload[parsedBytes:lenBytesEnd]))
		topicFilter := payload[lenBytesEnd : lenBytesEnd+topicLen]

		filterList = append(filterList, topicFilter)

		parsedBytes += topicLen + 2 // 2 bytes for topic length
	}

	if parsedBytes != payloadLen {
		return nil, errors.New("invalid payload")
	}

	return filterList, nil
}

func parseFixedHeaderFirstByte(b byte) (byte, string) {
	bs := bytes.ByteToBinaryString(b)
	return b >> 4, bs[4:] // packet type and flags bits (eg. 1101)
}
