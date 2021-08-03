// +build kots_experimental

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/replicatedhq/kots/pkg/apiserver"
	"github.com/replicatedhq/kots/pkg/cluster"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "run [upstream uri]",
		Short:         "Runs an application in an embedded cluster",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			slug := args[0]
			log := logger.NewCLILogger()
			log.Info("Running application %s", slug)

			loggerCtx := context.WithValue(context.Background(), "log", log)
			ctx, cancelFunc := context.WithCancel(loggerCtx)
			defer cancelFunc()

			// stat the kots api (aka, kotsadm in a former world)
			if err := startKotsadm(ctx, v.GetString("data-dir")); err != nil {
				return err
			}

			if err := startK8sAuthnz(ctx, v.GetString("data-dir")); err != nil {
				return err
			}

			// ensure data dir exist
			if _, err := os.Stat(v.GetString("data-dir")); os.IsNotExist(err) {
				if err := os.MkdirAll(v.GetString("data-dir"), 0755); err != nil {
					return err
				}
			}

			if err := cluster.Start(ctx, slug, v.GetString("data-dir")); err != nil {
				return err
			}

			// wait for interrupt, and stop the cluster when we receive
			log.Info("The cluster is running. Press ctrl+c to terminate")
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			cancelFunc()

			return nil
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cmd.Flags().String("data-dir", cwd, "directory to store admin console, kubernetes, and application data in")

	return cmd
}

func startKotsadm(ctx context.Context, dataDir string) error {
	filestore.ArchivesDir = filepath.Join(dataDir, "archives")

	util.PodNamespace = "default"
	util.KotsadmNamespace = "default"

	params := apiserver.APIServerParams{
		Version:                "???",
		SQLiteURI:              fmt.Sprintf("%s/kots.db", dataDir),
		AutocreateClusterToken: "TODO", // this needs to be static for an install, but different per installation
		EnableIdentity:         false,
	}

	go apiserver.Start(&params)

	return nil
}

func startK8sAuthnz(ctx context.Context, dataDir string) error {
	go cluster.StartAuthnzServer()

	return nil
}
