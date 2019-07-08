package types

// Session is a shared interface for Ship session structs
type Session interface {
	Output
	GetType() string
	GetRole() string
	GetName() string
	GetShipArgs() []string
	GetUploadURL() string
	GetNodeSelector() string
	GetCPULimit() string
	GetCPURequest() string
	GetMemoryLimit() string
	GetMemoryRequest() string
	GetParentWatchID() *string
	GetParentSequence() *int
}

// Output is a shared interface for ship structs that have the ability to produce ship tarballs
type Output interface {
	GetID() string
	GetWatchID() string
	GetUploadSequence() int
	GetS3Filepath() string
}
