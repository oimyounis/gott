package gott

import (
	"encoding/binary"
	"gott/bytes"
	"log"
)

var packetSeq = &sequencer{UpperBoundBits: 16, Start: 1}

func makeConnAckPacket(sessionPresent, returnCode byte) []byte {
	return []byte{TypeConnAck << 4, ConnectRemLen, sessionPresent, returnCode}
}

func makePubAckPacket(id []byte) []byte {
	return []byte{TypePubAck << 4, PubackRemLen, id[0], id[1]}
}

func makePubRecPacket(id []byte) []byte {
	return []byte{TypePubRec << 4, PubrecRemLen, id[0], id[1]}
}

func makePubRelPacket(id []byte) []byte {
	// +2 according to [MQTT-3.6.1-1]
	return []byte{TypePubRel<<4 + 2, PubrelRemLen, id[0], id[1]}
}

func makePubCompPacket(id []byte) []byte {
	return []byte{TypePubComp << 4, PubcompRemLen, id[0], id[1]}
}

func makeSubAckPacket(id []byte, filterList []filter) []byte {
	packet := []byte{TypeSubAck << 4}

	packet = append(packet, bytes.Encode(SubackRemLen+len(filterList))...)
	packet = append(packet, id...)

	for _, filter := range filterList {
		if validFilter(filter.Filter) {
			packet = append(packet, filter.QoS)
		} else {
			// in case of failure append SubackFailureCode (128)
			packet = append(packet, SubackFailureCode)
		}
	}
	log.Println("SUBACK", packet)
	return packet
}

func makeUnSubAckPacket(id []byte) []byte {
	packet := []byte{TypeUnsubAck << 4, UnsubackRemLen, id[0], id[1]}
	log.Println("UNSUBACK", packet)
	return packet
}

func makePingRespPacket() []byte {
	packet := []byte{TypePingResp << 4, PingrespRemLen}
	log.Println("PINGRESP", packet)
	return packet
}

func makePublishPacket(topic, payload []byte, dupFlag, qos, retainFlag byte) (packet []byte, packetID uint16) {
	if qos == 0 { // as per [MQTT-3.3.1-2]
		dupFlag = 0
	}
	if topic == nil {
		return
	}

	// 2nd byte... = remaining length (var header len + payload len)
	fixedHeader := []byte{TypePublish<<4 + dupFlag<<3 + qos<<1 + retainFlag}

	varHeader := make([]byte, 2) // topic name + packet identifier
	binary.BigEndian.PutUint16(varHeader, uint16(len(topic)))
	varHeader = append(varHeader, topic...)

	if qos > 0 {
		varHeader = append(varHeader, 0, 0)
		packetID = uint16(packetSeq.next())
		binary.BigEndian.PutUint16(varHeader[len(varHeader)-2:], packetID)
	}

	packet = append(packet, fixedHeader...)
	packet = append(packet, bytes.Encode(len(varHeader)+len(payload))...)
	packet = append(packet, varHeader...)
	packet = append(packet, payload...)

	return
}

func makePublishPacketWithID(packetID uint16, topic, payload []byte, dupFlag, qos, retainFlag byte) (packet []byte) {
	if qos == 0 { // as per [MQTT-3.3.1-2]
		dupFlag = 0
	}
	if topic == nil {
		return
	}

	// 2nd byte... = remaining length (var header len + payload len)
	fixedHeader := []byte{TypePublish<<4 + dupFlag<<3 + qos<<1 + retainFlag}

	varHeader := make([]byte, 2) // topic name + packet identifier
	binary.BigEndian.PutUint16(varHeader, uint16(len(topic)))
	varHeader = append(varHeader, topic...)

	if qos >= 1 {
		varHeader = append(varHeader, 0, 0)
		binary.BigEndian.PutUint16(varHeader[len(varHeader)-2:], packetID)
	}

	packet = append(packet, fixedHeader...)
	packet = append(packet, bytes.Encode(len(varHeader)+len(payload))...)
	packet = append(packet, varHeader...)
	packet = append(packet, payload...)

	return
}
