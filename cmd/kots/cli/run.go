// +build kots_experimental

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mholt/archiver"
	"github.com/replicatedhq/kots/pkg/apiserver"
	"github.com/replicatedhq/kots/pkg/cluster"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
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

			k8sLogRoot := filepath.Join(v.GetString("data-dir"), "kubernetes", "log")
			if err := os.MkdirAll(k8sLogRoot, 0755); err == nil {
				if f, err := ioutil.TempFile(k8sLogRoot, "k8s-"); err == nil {
					defer f.Close()
					grpclog.SetLoggerV2(grpclog.NewLoggerV2(f, f, f))
					klog.SetOutput(f)
					klog.LogToStderr(false)
				}
			}

			// TODO: @emosbaugh: im not sure i agree with this pattern. im not sure context is the best place for DI
			loggerCtx := context.WithValue(context.Background(), "log", log)
			ctx, cancelFunc := context.WithCancel(loggerCtx)
			defer cancelFunc()

			// this is here to ensure that the store is initialized before we spawn kots and kubernetes at the same time, which
			// might both try to initialize the store.
			_ = persistence.MustGetDBSession()
			persistence.SQLiteURI = fmt.Sprintf("%s/kots.db", v.GetString("data-dir")) // initialize here as well for the Authnz server to be able to use the store before kots comes up

			// ensure data dir exist
			if _, err := os.Stat(v.GetString("data-dir")); os.IsNotExist(err) {
				if err := os.MkdirAll(v.GetString("data-dir"), 0755); err != nil {
					return err
				}
			}

			if err := startK8sAuthnz(ctx, v.GetString("data-dir")); err != nil {
				return err
			}

			kubeconfigPath, err := cluster.Start(ctx, slug, v.GetString("data-dir"))
			if err != nil {
				return err
			}

			if err := ensureBinaries(v.GetString("data-dir")); err != nil {
				return err
			}

			// start the kots api (aka, kotsadm in a former world)
			if err := startKotsadm(ctx, v.GetString("data-dir"), v.GetString("shared-password"), kubeconfigPath); err != nil {
				return err
			}

			// wait for interrupt, and stop the cluster when we receive
			log.Info("The cluster is running. Press ctrl+c to terminate")
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			// TODO @emosbaugh: i dont think this works how you think it does. you probably have to catch another signal while the cluster terminates
			cancelFunc()

			return nil
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defaultDataDir := filepath.Join(cwd, "kotsdata")

	cmd.Flags().String("data-dir", defaultDataDir, "directory to store admin console, kubernetes, and application data in")
	cmd.Flags().String("shared-password", "", "the shared password to set to authenticate to the admin console")

	return cmd
}

func startKotsadm(ctx context.Context, dataDir string, sharedPassword string, kubeconfigPath string) error {
	filestore.ArchivesDir = filepath.Join(dataDir, "archives")

	// TODO @divolgin: something is odd about this pattern. these variables are set in two places to two different values, yet they are global.
	util.PodNamespace = "default"
	util.KotsadmTargetNamespace = "default"

	params := apiserver.APIServerParams{
		Version:                "???",
		SQLiteURI:              fmt.Sprintf("%s/kots.db", dataDir),
		AutocreateClusterToken: "TODO", // this needs to be static for an install, but different per installation
		EnableIdentity:         false,
		SharedPassword:         sharedPassword,
		KubeconfigPath:         kubeconfigPath,
		KotsDataDir:            dataDir,
	}

	go apiserver.Start(&params)

	return nil
}

func startK8sAuthnz(ctx context.Context, dataDir string) error {
	go cluster.StartAuthnzServer()

	return nil
}

func ensureBinaries(dataDir string) error {
	binariesRoot := filepath.Join(dataDir, "binaries")
	if _, err := os.Stat(binariesRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(binariesRoot, 0755); err != nil {
			return err
		}
	}

	if err := ensureKubectlBinary(binariesRoot); err != nil {
		return err
	}

	if err := ensureKustomizeBinary(binariesRoot); err != nil {
		return err
	}

	return nil
}

func ensureKubectlBinary(rootDir string) error {
	kubectlFilePath := filepath.Join(rootDir, "kubectl")
	if err := downloadFileFromURL(kubectlFilePath, "https://dl.k8s.io/release/v1.22.0/bin/linux/amd64/kubectl"); err != nil {
		return err
	}

	if err := os.Chmod(kubectlFilePath, 0755); err != nil {
		return err
	}

	return nil
}

func ensureKustomizeBinary(rootDir string) error {
	kustomizeArchive := filepath.Join(rootDir, "kustomize.tar.gz")
	if err := downloadFileFromURL(kustomizeArchive, "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz"); err != nil {
		return err
	}
	defer os.RemoveAll(kustomizeArchive)

	unarchived, err := ioutil.TempDir("", "kustomize")
	if err != nil {
		return err
	}
	defer os.RemoveAll(unarchived)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(kustomizeArchive, unarchived); err != nil {
		return err
	}

	err = os.Rename(filepath.Join(unarchived, "kustomize"), filepath.Join(rootDir, "kustomize3.5.4"))
	if err != nil {
		return err
	}

	return nil
}
