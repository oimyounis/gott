package gott

import (
	"encoding/binary"
	"gott/bytes"
	"log"
)

var packetSeq *Sequencer = &Sequencer{UpperBoundBits: 16, Start: 1}

func MakeConnAckPacket(sessionPresent, returnCode byte) []byte {
	return []byte{TYPE_CONNACK_BYTE, CONNECT_REM_LEN, sessionPresent, returnCode}
}

func MakePubAckPacket(id []byte) []byte {
	//binary.BigEndian.PutUint16(packet[2:], id)
	//log.Println("PUBACK", packet)
	return []byte{TYPE_PUBACK_BYTE, PUBACK_REM_LEN, id[0], id[1]}
}

func MakePubRecPacket(id []byte) []byte {
	//binary.BigEndian.PutUint16(packet[2:], id)
	//log.Println("PUBREC", packet)
	return []byte{TYPE_PUBREC_BYTE, PUBREC_REM_LEN, id[0], id[1]}
}

func MakePubCompPacket(id []byte) []byte {
	//binary.BigEndian.PutUint16(packet[2:], id)
	//log.Println("PUBCOMP", packet)
	return []byte{TYPE_PUBCOMP_BYTE, PUBCOMP_REM_LEN, id[0], id[1]}
}

func MakeSubAckPacket(id []byte, filterList []Filter) []byte {
	packet := []byte{TYPE_SUBACK_BYTE}

	packet = append(packet, bytes.Encode(SUBACK_REM_LEN+len(filterList))...)
	packet = append(packet, id...)

	for _, filter := range filterList {
		if ValidFilter(filter.Filter) {
			packet = append(packet, filter.QoS) // QoS here should be the Maximum QoS determined by the server, see [3.8.4]
		} else {
			// in case of failure append SUBACK_FAILURE_CODE (128)
			packet = append(packet, SUBACK_FAILURE_CODE)
		}
	}
	log.Println("SUBACK", packet)
	return packet
}

func MakeUnSubAckPacket(id []byte) []byte {
	packet := []byte{TYPE_UNSUBACK_BYTE, UNSUBACK_REM_LEN, id[0], id[1]}
	log.Println("UNSUBACK", packet)
	return packet
}

func MakePingRespPacket() []byte {
	packet := []byte{TYPE_PINGRESP_BYTE, PINGRESP_REM_LEN}
	log.Println("PINGRESP", packet)
	return packet
}

// Not completed yet
func MakePublishPacket(topic, payload []byte, dupFlag, qos, retainFlag byte) (packet []byte) {
	// TODO: complete the implementation of this func
	if qos == 0 { // as per [MQTT-3.3.1-2]
		dupFlag = 0
	}
	if topic == nil {
		return
	}

	// 2nd byte... = remaining length (var header len + payload len)
	fixedHeader := []byte{TYPE_PUBLISH_BYTE + dupFlag<<3 + qos<<1 + retainFlag}

	varHeader := make([]byte, 2) // topic name + packet identifier
	binary.BigEndian.PutUint16(varHeader, uint16(len(topic)))
	varHeader = append(varHeader, topic...)

	if qos >= 1 {
		varHeader = append(varHeader, 0, 0)
		binary.BigEndian.PutUint16(varHeader[len(varHeader)-2:], uint16(packetSeq.Next()))
	}

	packet = append(packet, fixedHeader...)
	packet = append(packet, bytes.Encode(len(varHeader)+len(payload))...)
	packet = append(packet, varHeader...)
	packet = append(packet, payload...)

	return
}
