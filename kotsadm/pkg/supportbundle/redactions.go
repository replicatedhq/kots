package supportbundle

import (
	"fmt"

	"github.com/replicatedhq/troubleshoot/pkg/redact"
)

// todo: actually persist this in db
var redactions map[string]redact.RedactionList

func GetRedactions(bundleID string) (redact.RedactionList, error) {
	redacts, ok := redactions[bundleID]
	if !ok {
		return redact.RedactionList{}, fmt.Errorf("unable to find redactions for bundle %s", bundleID)
	}
	return redacts, nil
}

func SetRedactions(bundleID string, redacts redact.RedactionList) error {
	if redactions == nil {
		redactions = map[string]redact.RedactionList{}
	}
	if _, ok := redactions[bundleID]; ok {
		// overwriting previously stored value is an error
		return fmt.Errorf("redactions for bundle %s already present", bundleID)
	}
	redactions[bundleID] = redacts
	return nil
}
