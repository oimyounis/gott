package gott

// Packet types.
const (
	TypeReserved = iota
	TypeConnect
	TypeConnAck
	TypePublish
	TypePubAck
	TypePubRec
	TypePubRel
	TypePubComp
	TypeSubscribe
	TypeSubAck
	TypeUnsubscribe
	TypeUnsubAck
	TypePingReq
	TypePingResp
	TypeDisconnect
)

// Remaining lengths according to the MQTT spec.
const (
	ConnectRemLen       = 2 // 2 is constant remaining len as per [3.2.1]
	ConnectVarHeaderLen = 10
	PubackRemLen        = 2 // 2 is constant remaining len as per [3.4.1]
	PubrecRemLen        = 2 // 2 is constant remaining len as per [3.5.1]
	PubrelRemLen        = 2 // 2 is constant remaining len as per [3.6.1]
	PubcompRemLen       = 2 // 2 is constant remaining len as per [3.7.1]
	SubackRemLen        = 2 // 2 is constant remaining len as per [3.9.2]
	UnsubackRemLen      = 2 // 2 is constant remaining len as per [3.11.1]
	PingrespRemLen      = 0 // 0 is constant remaining len as per [3.13.1]
)

// Subscribe Ack return codes.
const (
	SubackFailureCode byte = 128
)

// Topic specific constants.
var (
	TopicDelim               = []byte{47} // /
	TopicSingleLevelWildcard = []byte{43} // +
	TopicMultiLevelWildcard  = []byte{35} // #
)

// Application Message statuses.
const (
	StatusUnacknowledged byte = iota
	StatusPubackReceived
	StatusPubrecReceived
	StatusPubrelReceived
	StatusPubcompReceived
)

const (
	pluginDir            = "plugins"
)
