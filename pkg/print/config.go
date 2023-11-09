package print

import (
	"fmt"
	"strings"

	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/logger"
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
