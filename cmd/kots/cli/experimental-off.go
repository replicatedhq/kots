//go:build !kots_experimental
// +build !kots_experimental

package cli

import "github.com/spf13/cobra"

func addExperimentalCmds(cmd *cobra.Command) {
	// no-op
}
