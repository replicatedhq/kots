package cli

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type VersionOutput struct {
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion,omitempty"`
	InstallLatest string `json:"installLatest,omitempty"`
}

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the current version and exit",
		Long:  `Print the current version and exit`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r := mux.NewRouter()

			spa := handlers.SPAHandler{}
			r.PathPrefix("/").Handler(spa)

			srv := &http.Server{
				Handler: r,
				Addr:    ":30888",
			}

			fmt.Printf("Starting KOTS SPA handler on port %d...\n", 30888)

			log.Fatal(srv.ListenAndServe())

			return nil
		},
	}

	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	return cmd
}
