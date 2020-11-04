package types

type ErrorTimeout struct {
	Message string
}

func (e *ErrorTimeout) Error() string {
	return e.Message
}

type ErrorAppsRestore struct {
	Message string
}

func (e *ErrorAppsRestore) Error() string {
	return e.Message
}
