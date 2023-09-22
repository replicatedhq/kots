package print

import (
	"fmt"
	"strings"

	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
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

func PreflightErrors(log *logger.CLILogger, results []*preflight.UploadPreflightResult) {
	var s strings.Builder
	s.WriteString("\nPreflight check results (state - title - message)")
	for _, result := range results {
		s.WriteString(fmt.Sprintf("\n%s - %q - %q", preflightState(result), result.Title, result.Message))
	}
	log.Info(s.String())
}

func preflightState(r *preflight.UploadPreflightResult) string {
	if r.IsFail {
		return "FAIL"
	}
	if r.IsWarn {
		return "WARN"
	}
	if r.IsPass {
		return "PASS"
	}
	// We should never get here
	return "UNKNOWN"
}
