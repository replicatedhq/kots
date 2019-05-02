package debug

import (
	"expvar"
	"strconv"
)

var (
	status *expvar.String
)

func Init() {
	status = expvar.NewString("ok")
	SetStatus(true)
}

func SetStatus(b bool) {
	status.Set(strconv.FormatBool(b))
}
