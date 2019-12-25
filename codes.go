package gott

// Connect Ack return codes
const (
	ConnectAccepted byte = iota
	ConnectUnacceptableProto
	ConnectIDRejected
	ConnectServerUnavailable
	ConnectBadUsernamePassword
	ConnectNotAuthorized
)
