package gott

const (
	TYPE_RESERVED = iota
	TYPE_CONNECT
	TYPE_CONNACK
	TYPE_PUBLISH
	TYPE_PUBACK
	TYPE_PUBREC
	TYPE_PUBREL
	TYPE_PUBCOMP
	TYPE_SUBSCRIBE
	TYPE_SUBACK
	TYPE_UNSUBSCRIBE
	TYPE_UNSUBACK
	TYPE_PINGREQ
	TYPE_PINGRESP
	TYPE_DISCONNECT
)

const (
	TYPE_RESERVED_BYTE    byte = TYPE_RESERVED << 4
	TYPE_CONNECT_BYTE          = TYPE_CONNECT << 4
	TYPE_CONNACK_BYTE          = TYPE_CONNACK << 4
	TYPE_PUBLISH_BYTE          = TYPE_PUBLISH << 4
	TYPE_PUBACK_BYTE           = TYPE_PUBACK << 4
	TYPE_PUBREC_BYTE           = TYPE_PUBREC << 4
	TYPE_PUBREL_BYTE           = TYPE_PUBREL << 4
	TYPE_PUBCOMP_BYTE          = TYPE_PUBCOMP << 4
	TYPE_SUBSCRIBE_BYTE        = TYPE_SUBSCRIBE << 4
	TYPE_SUBACK_BYTE           = TYPE_SUBACK << 4
	TYPE_UNSUBSCRIBE_BYTE      = TYPE_UNSUBSCRIBE << 4
	TYPE_UNSUBACK_BYTE         = TYPE_UNSUBACK << 4
	TYPE_PINGREQ_BYTE          = TYPE_PINGREQ << 4
	TYPE_PINGRESP_BYTE         = TYPE_PINGRESP << 4
	TYPE_DISCONNECT_BYTE       = TYPE_DISCONNECT << 4
)

const (
	CONNECT_REM_LEN        = 2 // 2 is constant remaining len as per [3.2.1]
	CONNECT_VAR_HEADER_LEN = 10
	PUBACK_REM_LEN         = 2 // 2 is constant remaining len as per [3.4.1]
	PUBREC_REM_LEN         = 2 // 2 is constant remaining len as per [3.5.1]
	PUBREL_REM_LEN         = 2 // 2 is constant remaining len as per [3.6.1]
	PUBCOMP_REM_LEN        = 2 // 2 is constant remaining len as per [3.7.1]
	SUBACK_REM_LEN         = 2 // 2 is constant remaining len as per [3.9.2]
	UNSUBACK_REM_LEN       = 2 // 2 is constant remaining len as per [3.11.1]
	PINGRESP_REM_LEN       = 0 // 0 is constant remaining len as per [3.13.1]
)

const (
	SUBACK_FAILURE_CODE byte = 128
)

var (
	TOPIC_DELIM                 = []byte{47} // /
	TOPIC_SINGLE_LEVEL_WILDCARD = []byte{43} // +
	TOPIC_MULTI_LEVEL_WILDCARD  = []byte{35} // #
)

const (
	STATUS_UNACKNOWLEDGED byte = iota
	STATUS_PUBACK_RECEIVED
	STATUS_PUBREC_RECEIVED
	STATUS_PUBREL_RECEIVED
	STATUS_PUBCOMP_RECEIVED
)
