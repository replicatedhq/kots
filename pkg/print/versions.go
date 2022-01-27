package print

import (
	"encoding/json"
	"fmt"
	"time"
)

type AppVersionResponse struct {
	VersionLabel string     `json:"versionLabel"`
	Sequence     int64      `json:"sequence"`
	CreatedOn    time.Time  `json:"createdOn"`
	Status       string     `json:"status"`
	DeployedAt   *time.Time `json:"deployedAt"`
	Source       string     `json:"source"`
}

func Versions(versions []AppVersionResponse, format string) {
	switch format {
	case "json":
		printVersionsJSON(versions)
	default:
		printVersionsTable(versions)
	}
}

func printVersionsJSON(versions []AppVersionResponse) {
	str, _ := json.MarshalIndent(versions, "", "    ")
	fmt.Println(string(str))
}

func printVersionsTable(versions []AppVersionResponse) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%v\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "VERSION", "SEQUENCE", "STATUS", "SOURCE")
	for _, version := range versions {
		fmt.Fprintf(w, fmtColumns, version.VersionLabel, version.Sequence, version.Status, version.Source)
	}
}
