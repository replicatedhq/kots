package print

import (
	"fmt"
	"strings"

	"github.com/replicatedhq/kots/pkg/preflight"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
)

func PreflightResults(results preflighttypes.PreflightResults) {
	w := NewTabWriter()
	defer w.Flush()

	if len(results.Errors) > 0 {
		fmt.Fprintf(w, "\n")
		for _, err := range results.Errors {
			fmtColumns := "	- %s\n"
			fmt.Fprintf(w, fmtColumns, err.Error)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(results.Results) > 0 {
		fmt.Fprintf(w, "\n")
		fmtColumns := "%s\t%s\t%s\n"
		fmt.Fprintf(w, fmtColumns, "STATE", "TITLE", "MESSAGE")
		for _, result := range results.Results {
			fmt.Fprintf(w, fmtColumns, strings.ToUpper(preflight.GetPreflightCheckState(result)), result.Title, result.Message)
		}
		fmt.Fprintf(w, "\n")
	}
}
