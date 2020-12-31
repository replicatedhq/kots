package print

import (
	"encoding/json"
	"fmt"
)

type App struct {
	Slug  string `json:"slug"`
	State string `json:"state"`
}

func Apps(apps []App, format string) {
	switch format {
	case "json":
		printAppsJSON(apps)
	default:
		printAppsTable(apps)
	}
}

func printAppsJSON(apps []App) {
	str, _ := json.MarshalIndent(apps, "", "    ")
	fmt.Println(string(str))
}

func printAppsTable(apps []App) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "SLUG", "STATUS")
	for _, app := range apps {
		fmt.Fprintf(w, fmtColumns, app.Slug, app.State)
	}
}
