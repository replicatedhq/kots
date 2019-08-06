package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kotsadm-operator",
		Short: "Run the kotsadm operator component in cluster",
		Long: `kotsadm-operator is akin to "gitops lite". this operator provides the change management control
of a gitops pipeline without setting up a full end-to-end gitops delivery process`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// v := viper.GetViper()
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.Flags().String("kotsadm-uri", "http://kotsadm:8880", "the endpoint of the kotsadm api server to connect to")
	cmd.Flags().String("kueconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use when connecting to the cluster")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("KOTSADM")
	viper.AutomaticEnv()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
