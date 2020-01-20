package gott

const (
	mqttv311 = 4
)

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
	defaultConfigContent = `# GOTT configuration file

# listen property is the address that the broker will listen on,
# in the format hostname_or_ip:port.
# In case you wanted to enable connections over TLS only, leave this empty to
# disable it.
# Default is ":1883".
listen: ":1883"

# tls property defines TLS configurations.
# To disable leave any of the child properties empty.
# tls.listen: Defines the address that the broker will use to serve traffic over
# TLS, in the format hostname_or_ip:port, default is ":8883".
# tls.cert: Absolute path to the certificate file.
# tls.key: Absolute path to the key file.
# Disabled by default.
tls:
  listen: ":8883"
  cert: ""
  key: ""

# log_level property defines the minimum level to which the broker should log messages,
# available levels are "debug", "info", "error" and "fatal",
# "debug" is the lowest and "fatal" is the highest,
# each level includes higher levels as well, default is "error".
log_level: "error"

# plugins property is a collection of plugin names,
# all plugins listed here must be placed in the plugins directory to be loaded,
# plugins are loaded by the order they were listed in.
plugins:
#  - myplugin.so
`
)
