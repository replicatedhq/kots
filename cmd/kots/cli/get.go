package cli

import (
	"os"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	handlertypes "github.com/replicatedhq/kots/pkg/api/handlers/types"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
)

func GetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [resource]",
		Short: "Display kots resources",
		Long: `Examples:
kubectl kots get apps`,

		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.Help()
				os.Exit(1)
			}

			switch args[0] {
			case "backup", "backups":
				err := getBackupsCmd(cmd, args)
				return errors.Wrap(err, "failed to get backups")
			case "restore", "restores":
				err := getRestoresCmd(cmd, args)
				return errors.Wrap(err, "failed to get restores")
			case "app", "apps":
				err := getAppsCmd(cmd, args)
				return errors.Wrap(err, "failed to get apps")
			default:
				cmd.Help()
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringP("output", "o", "", "Output format. Supported values: json")

	return cmd
}

func getBackupsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	options := snapshot.ListInstanceBackupsOptions{
		Namespace: v.GetString("namespace"),
	}
	backups, err := snapshot.ListInstanceBackups(options)
	if err != nil {
		return errors.Wrap(err, "failed to list instance backups")
	}

	print.Backups(backups)

	return nil
}

func getRestoresCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	options := snapshot.ListInstanceRestoresOptions{
		Namespace: v.GetString("namespace"),
	}
	restores, err := snapshot.ListInstanceRestores(options)
	if err != nil {
		return errors.Wrap(err, "failed to list instance restores")
	}

	print.Restores(restores)

	return nil
}

func getAppsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	log := logger.NewCLILogger()

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace := v.GetString("namespace")
	if err := validateNamespace(namespace); err != nil {
		return errors.Wrap(err, "failed to validate namespace")
	}

	podName, err := k8sutil.FindKotsadm(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	localPort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 0, 3000, namespace, podName, false, stopCh, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to start port forwarding")
	}

	go func() {
		select {
		case err := <-errChan:
			if err != nil {
				log.Error(err)
			}
		case <-stopCh:
		}
	}()

	authSlug, err := auth.GetOrCreateAuthSlug(kubernetesConfigFlags, namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", namespace)
		if v.GetBool("debug") {
			return errors.Wrap(err, "failed to get kotsadm auth slug")
		}
		os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/apps", localPort)
	apps, err := getApps(url, authSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get apps")
	}

	printableApps := make([]print.App, 0)
	for _, app := range apps.Apps {
		url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/status", localPort, app.Slug)
		appStatus, err := getAppStatus(url, authSlug)
		if err != nil {
			return errors.Wrapf(err, "failed to get app status for %s", app.Slug)
		}
		printableApps = append(printableApps, print.App{
			Slug:  app.Slug,
			State: string(appStatus.AppStatus.State),
		})
	}

	print.Apps(printableApps, v.GetString("output"))

	return nil
}

func getApps(url string, authSlug string) (*handlertypes.ListAppsResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	apps := &handlertypes.ListAppsResponse{}
	if err := json.Unmarshal(b, apps); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal apps")
	}

	return apps, nil
}

func getAppStatus(url string, authSlug string) (*handlertypes.AppStatusResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	status := &handlertypes.AppStatusResponse{}
	if err := json.Unmarshal(b, status); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal status")
	}

	return status, nil
}
