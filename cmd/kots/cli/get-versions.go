package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	cmd.Flags().Int("current-page", 0, "offset by page size at which to start retrieving versions")
	cmd.Flags().Int("page-size", 20, "number of versions to return (defaults to 20)")
	cmd.Flags().Bool("pin-latest", false, "set to true to always return the latest version at the beginning")
	cmd.Flags().Bool("pin-latest-deployable", false, "set to true to always return the latest deployable version at the beginning")
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

	urlVals := url.Values{}
	urlVals.Set("currentPage", fmt.Sprintf("%d", v.GetInt("current-page")))
	urlVals.Set("pageSize", fmt.Sprintf("%d", v.GetInt("page-size")))
	urlVals.Set("pinLatest", fmt.Sprintf("%t", v.GetBool("pin-latest")))
	urlVals.Set("pinLatestDeployable", fmt.Sprintf("%t", v.GetBool("pin-latest-deployable")))

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/versions?%s", localPort, url.PathEscape(appSlug), urlVals.Encode())
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

func getAppVersions(url string, authSlug string) (*handlers.GetAppVersionHistoryResponse, error) {
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

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	appVersions := handlers.GetAppVersionHistoryResponse{}
	if err := json.Unmarshal(b, &appVersions); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal app versions")
	}

	return &appVersions, nil
}
