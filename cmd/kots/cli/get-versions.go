package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "versions [appSlug]",
		Short:         "Get App Versions",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getVersionsCmd,
	}

	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	return cmd
}

func getVersionsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	if len(args) == 0 {
		cmd.Help()
		os.Exit(1)
	}

	appSlug := args[0]

	output := v.GetString("output")
	if output != "json" && output != "" {
		return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
	}

	log := logger.NewCLILogger()

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
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

	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, podName, false, stopCh, log)
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

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/versions", localPort, appSlug)
	appVersions, err := getAppVersions(url, authSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get app versions")
	}

	appVersionResponse := []print.AppVersionResponse{}

	for _, version := range appVersions.VersionHistory {
		response := print.AppVersionResponse{
			VersionLabel: version.VersionLabel,
			Sequence:     version.Sequence,
			CreatedOn:    *version.CreatedOn,
			DeployedAt:   version.DeployedAt,
			Status:       string(version.Status),
			Source:       version.Source,
		}

		appVersionResponse = append(appVersionResponse, response)
	}

	print.Versions(appVersionResponse, output)

	return nil
}

func getAppVersions(url string, authSlug string) (*handlers.GetAppVersionsResponse, error) {
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

	if resp.StatusCode == 500 {
		return nil, fmt.Errorf("check the app slug")
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	appVersions := handlers.GetAppVersionsResponse{}
	if err := json.Unmarshal(b, &appVersions); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal apps")
	}

	return &appVersions, nil
}
