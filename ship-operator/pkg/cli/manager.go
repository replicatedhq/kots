package cli

import (
	"log"

	"github.com/replicatedhq/ship-operator/pkg/apis"
	"github.com/replicatedhq/ship-operator/pkg/controller"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func Manager() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "starts the ship-operator manager",
		Run: func(cmd *cobra.Command, args []string) {

			// Get a config to talk to the apiserver
			cfg, err := config.GetConfig()
			if err != nil {
				log.Fatal(err)
			}

			// Create a new Cmd to provide shared dependencies and start components
			mgr, err := manager.New(cfg, manager.Options{})
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("Registering Components.")

			// Setup Scheme for all resources
			if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
				log.Fatal(err)
			}

			// Setup all Controllers
			if err := controller.AddToManager(mgr); err != nil {
				log.Fatal(err)
			}

			log.Printf("Starting the Cmd.")

			// Start the Cmd
			log.Fatal(mgr.Start(signals.SetupSignalHandler()))
		},
	}

	return cmd
}
