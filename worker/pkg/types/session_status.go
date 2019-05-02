package types

var (
	SessionStatusReady    SessionStatus = "ready"
	SessionStatusRunning  SessionStatus = "running"
	SessionStatusComplete SessionStatus = "complete"
	SessionStatusFailed   SessionStatus = "failed"
)

type SessionStatus string
