package print

import (
	"github.com/fatih/color"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

func ConfigValidationErrors(log *logger.CLILogger, groupValidationErrors []configtypes.ConfigGroupValidationError) {
	errPrint := color.New(color.FgHiRed)
	errPrint.Println("Following config items have validation errors:")
	for _, groupValidationError := range groupValidationErrors {
		errPrint.Printf("Group - %s\n", groupValidationError.Name)
		errPrint.Println("  Items:")
		for _, itemValidationError := range groupValidationError.ItemErrors {
			errPrint.Printf("    Name: %s\n", itemValidationError.Name)
			errPrint.Println("    Errors:")
			for _, validationError := range itemValidationError.ValidationErrors {
				errPrint.Printf("      - %s\n", validationError.Message)
			}
		}
	}
}
