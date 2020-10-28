package print

import (
	"os"
	"text/tabwriter"
)

const (
	minWidth = 0
	tabWidth = 8
	padding  = 4
	padChar  = ' '
)

func NewTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, minWidth, tabWidth, padding, padChar, tabwriter.TabIndent)
}
