package errors

// FetchFilesError is used by implementations of FileFetcher to represent an error
// fetching the upstream resource
type FetchFilesError struct {
	Message string
}

func (f FetchFilesError) Error() string {
	return f.Message
}
