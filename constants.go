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
	StatusUnacknowledged int32 = iota
	StatusPubackReceived
	StatusPubrecReceived
	StatusPubrelReceived
	StatusPubcompReceived
)

const (
	pluginDir            = "plugins"
	defaultConfigContent = `# GOTT Configuration File

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

# To disable MQTT over non-TLS WebSockets set 'listen' to an empty string.
# To allow all Origins leave 'origins' empty.
# If 'reject_empty_origin' is set to true, any request with a missing or empty Origin header
# will be rejected.
# 'wss' holds the configuration to enable WebSockets over TLS,
  # To disable, leave any of the child properties empty.
  # Disabled by default.
websockets:
  listen: ":8083"
  path: "/ws"
  wss:
    listen: ":8084"
    cert: ""
    key: ""
  reject_empty_origin: false
  origins:
    # - https://website.com
    # - http://localhost:8000

# To disable the dashboard set 'listen' to an empty string.
# To disable the dashboard over TLS set any of the child properties to an empty string.
dashboard:
  listen: ":18883"
  tls:
    listen: ":18884"
    cert: ""
    key: ""

# logging property adjusts how the logger should behave.
  # logging.log_level: Defines the minimum level to which the broker should log messages,
    # available levels are "debug", "info", "error" and "fatal",
    # "debug" is the lowest and "fatal" is the highest,
    # each level includes higher levels as well, default is "error".
  # logging.filename: The name to use for the log file.
  # logging.max_size: The maximum size in megabytes of the log file to trigger rotation.
  # logging.max_backups: The maximum number of log files to keep, when the max is reached,
    # the logger will start deleting older backups.
  # logging.max_age: The number of days that the logs will live before deleted.
  # logging.enable_compression: Indicates whether to compress log files when the max_size
  # is reached or not. (highly recommended to be enabled to save disk space)
logging:
  log_level: "error"
  filename: "gott.log"
  max_size: 5 # megabytes
  max_backups: 20
  max_age: 30 # days
  enable_compression: true

# plugins property is a collection of plugin names,
# all plugins listed here must be placed in the plugins directory to be loaded,
# plugins are loaded by the order they were listed in.
plugins:
#  - myplugin.so
`
)
