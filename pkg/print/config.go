package print

import (
	"fmt"
	"strings"

	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	tsPreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

func ConfigValidationErrors(log *logger.CLILogger, groupValidationErrors []configtypes.ConfigGroupValidationError) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Following config items have validation errors:\n\n")
	for _, groupValidationError := range groupValidationErrors {
		fmt.Fprintf(&sb, "Group: %s\n", groupValidationError.Name)
		fmt.Fprintf(&sb, "  Items:\n")
		for _, itemValidationError := range groupValidationError.ItemErrors {
			fmt.Fprintf(&sb, "    Name: %s\n", itemValidationError.Name)
			fmt.Fprintf(&sb, "    Errors:\n")
			for _, validationError := range itemValidationError.ValidationErrors {
				fmt.Fprintf(&sb, "      - %s\n", validationError.Message)
			}
		}
	}

	log.FinishSpinnerWithError()
	log.Errorf(sb.String())
}

func PreflightErrors(log *logger.CLILogger, results []*tsPreflight.UploadPreflightResult) {
	w := NewTabWriter()
	defer w.Flush()

	fmt.Fprintf(w, "\n")
	fmtColumns := "%s\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "STATE", "TITLE", "MESSAGE")
	for _, result := range results {
		fmt.Fprintf(w, fmtColumns, strings.ToUpper(preflight.GetPreflightCheckState(result)), result.Title, result.Message)
	}
	fmt.Fprintf(w, "\n")
}
