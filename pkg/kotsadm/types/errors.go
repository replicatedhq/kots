package types

type ErrorAppsRestore struct {
	Message string
}

func (e *ErrorAppsRestore) Error() string {
	return e.Message
}
