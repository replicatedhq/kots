package supportbundle

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/troubleshoot/pkg/redact"
)

func GetRedactions(bundleID string) (redact.RedactionList, error) {
	db := persistence.MustGetPGSession()
	q := `select redact_report from supportbundle where id = $1`

	var redactString sql.NullString
	row := db.QueryRow(q, bundleID)
	err := row.Scan(&redactString)
	if err != nil {
		return redact.RedactionList{}, errors.Wrap(err, "select redact_report")
	}

	if !redactString.Valid || redactString.String == "" {
		return redact.RedactionList{}, fmt.Errorf("unable to find redactions for bundle %s", bundleID)
	}

	redacts := redact.RedactionList{}
	err = json.Unmarshal([]byte(redactString.String), &redacts)
	if err != nil {
		return redact.RedactionList{}, errors.Wrap(err, "unmarshal redact report")
	}

	return redacts, nil
}

func SetRedactions(bundleID string, redacts redact.RedactionList) error {
	db := persistence.MustGetPGSession()

	redactBytes, err := json.Marshal(redacts)
	if err != nil {
		return errors.Wrap(err, "marshal redactionlist")
	}

	query := `update supportbundle set redact_report = $1 where id = $2`
	_, err = db.Exec(query, string(redactBytes), bundleID)
	if err != nil {
		return errors.Wrap(err, "failed to set support bundle redact report")
	}
	return nil
}
