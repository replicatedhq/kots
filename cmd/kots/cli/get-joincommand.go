package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetJoinCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:           "join-command",
		Short:         "Get embedded cluster join command",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		Hidden:        true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return fmt.Errorf("failed to get clientset: %w", err)
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			joinCmd, err := getJoinCommandCmd(cmd.Context(), clientset, namespace)
			if err != nil {
				return err
			}

			if format == "string" {
				fmt.Println(strings.Join(joinCmd, " "))
				return nil
			} else if format == "json" {
				type joinCommandResponse struct {
					Command []string `json:"command"`
				}
				joinCmdResponse := joinCommandResponse{
					Command: joinCmd,
				}
				b, err := json.Marshal(joinCmdResponse)
				if err != nil {
					return fmt.Errorf("failed to marshal join command: %w", err)
				}
				fmt.Println(string(b))
				return nil
			}

			return fmt.Errorf("invalid output format: %s", format)
		},
	}

	cmd.Flags().StringVarP(&format, "output", "o", "string", "Output format. One of: string, json")

	return cmd
}

func getJoinCommandCmd(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]string, error) {
	// determine the IP address and port of the kotsadm service
	// this only runs inside an embedded cluster and so we don't need to setup port forwarding
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, "kotsadm", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get kotsadm service: %w", err)
	}
	kotsadmIP := svc.Spec.ClusterIP
	if kotsadmIP == "" {
		return nil, fmt.Errorf("kotsadm service ip was empty")
	}

	if len(svc.Spec.Ports) == 0 {
		return nil, fmt.Errorf("kotsadm service ports were empty")
	}
	kotsadmPort := svc.Spec.Ports[0].Port

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get kotsadm auth slug: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/embedded-cluster/roles", kotsadmIP, kotsadmPort)
	roles, err := getRoles(url, authSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	controllerRole := roles.ControllerRoleName
	if controllerRole == "" && len(roles.Roles) > 0 {
		controllerRole = roles.Roles[0]
	}
	if controllerRole == "" {
		return nil, fmt.Errorf("unable to determine controller role name")
	}

	// get a join command with the controller role with a post to /api/v1/embedded-cluster/generate-node-join-command
	url = fmt.Sprintf("http://%s:%d/api/v1/embedded-cluster/generate-node-join-command", kotsadmIP, kotsadmPort)
	joinCommand, err := getJoinCommand(url, authSlug, []string{controllerRole})
	if err != nil {
		return nil, fmt.Errorf("failed to get join command: %w", err)
	}

	return joinCommand.Command, nil
}

// determine the embedded cluster roles list from /api/v1/embedded-cluster/roles
func getRoles(url string, authSlug string) (*types.GetEmbeddedClusterRolesResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	roles := &types.GetEmbeddedClusterRolesResponse{}
	if err := json.Unmarshal(b, roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}

	return roles, nil
}

func getJoinCommand(url string, authSlug string, roles []string) (*types.GenerateEmbeddedClusterNodeJoinCommandResponse, error) {
	payload := types.GenerateEmbeddedClusterNodeJoinCommandRequest{
		Roles: roles,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal roles: %w", err)
	}

	newReq, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fullResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	joinCommand := &types.GenerateEmbeddedClusterNodeJoinCommandResponse{}
	if err := json.Unmarshal(fullResponse, joinCommand); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}

	return joinCommand, nil
}
