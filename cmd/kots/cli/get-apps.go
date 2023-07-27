package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "apps",
		Aliases:       []string{"app"},
		Short:         "Get apps",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getAppsCmd,
	}

	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")

	return cmd
}

func getAppsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	log := logger.NewCLILogger(cmd.OutOrStdout())

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
	if err != nil {
		return errors.Wrap(err, "failed to get namespace")
	}

	getPodName := func() (string, error) {
		return k8sutil.FindKotsadm(clientset, namespace)
	}

	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
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

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
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
		versionLabel := ""
		if app.Downstream.CurrentVersion != nil {
			versionLabel = app.Downstream.CurrentVersion.VersionLabel
		}
		url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/status", localPort, app.Slug)
		appStatus, err := getAppStatus(url, authSlug)
		if err != nil {
			return errors.Wrapf(err, "failed to get app status for %s", app.Slug)
		}
		printableApps = append(printableApps, print.App{
			Slug:         app.Slug,
			State:        string(appStatus.AppStatus.State),
			VersionLabel: versionLabel,
		})
	}

	print.Apps(printableApps, v.GetString("output"))

	return nil
}

func getApps(url string, authSlug string) (*types.ListAppsResponse, error) {
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	apps := &types.ListAppsResponse{}
	if err := json.Unmarshal(b, apps); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal apps")
	}

	return apps, nil
}

func getAppStatus(url string, authSlug string) (*types.AppStatusResponse, error) {
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	status := &types.AppStatusResponse{}
	if err := json.Unmarshal(b, status); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal status")
	}

	return status, nil
}
