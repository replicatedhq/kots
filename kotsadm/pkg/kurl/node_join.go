package kurl

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"k8s.io/client-go/kubernetes"
)

var (
	// ConfigExpirationRegeneratePeriod the expiry grace period to regenerate the bootstrap token
	ConfigExpirationRegeneratePeriod = 10 * time.Minute
)

// GenerateAddNodeCommand will generate the Kurl node add command for a master or worker
func GenerateAddNodeCommand(client kubernetes.Interface, master bool) ([]string, *time.Time, error) {
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, nil, errors.Wrap(err, "get kubernetes server version")
	}

	cm, err := ReadConfigMap(client)
	if err != nil {
		return nil, nil, errors.Wrap(err, "read kurl configmap")
	}

	cm, err = UpdateConfigMap(client, shouldRegenerateBootstrapToken(cm.Data), shouldUploadCerts(cm.Data, master))
	if err != nil {
		return nil, nil, errors.Wrap(err, "update kurl configmap")
	}

	data := cm.Data

	bootstrapTokenExpiration, err := time.Parse(time.RFC3339, data[bootstrapTokenExpirationKey])
	if err != nil {
		return nil, nil, errors.Wrap(err, "get bootstrap token expiration")
	}

	var command []string

	if ok, _ := strconv.ParseBool(data["airgap"]); ok {
		command = append(command, "cat join.sh | sudo bash -s airgap")
	} else {
		command = append(command, fmt.Sprintf("curl -sSL %s/%s/join.sh | sudo bash -s", data["kurl_url"], data["installer_id"]))
	}

	command = append(command,
		fmt.Sprintf("kubernetes-master-address=%s", data["kubernetes_api_address"]),
		fmt.Sprintf("kubeadm-token=%s", data["bootstrap_token"]),
		fmt.Sprintf("kubeadm-token-ca-hash=%s", data["ca_hash"]),
		fmt.Sprintf("docker-registry-ip=%s", data["docker_registry_ip"]),
		fmt.Sprintf("kubernetes-version=%s", versionInfo.GitVersion),
	)

	if master {
		command = append(command,
			fmt.Sprintf("cert-key=%s", data["cert_key"]),
			"control-plane",
		)
	}

	return command, &bootstrapTokenExpiration, nil
}

func shouldRegenerateBootstrapToken(data map[string]string) bool {
	value, ok := data[bootstrapTokenExpirationKey]
	if !ok {
		return true
	}

	bootstrapTokenExpiration, err := time.Parse(time.RFC3339, value)
	if err != nil {
		logger.Debugf("Failed to parse bootstrap_token_expiration %q: %v", value, err)
		return true
	}

	if time.Now().Add(ConfigExpirationRegeneratePeriod).After(bootstrapTokenExpiration) {
		logger.Debugf("Bootstrap token expired %s, regenerating", bootstrapTokenExpiration)
		return true
	}
	return false
}

func shouldUploadCerts(data map[string]string, master bool) bool {
	if !master {
		return false
	}

	value, ok := data[certsExpirationKey]
	if !ok {
		return true
	}

	uploadCertsExpiration, err := time.Parse(time.RFC3339, value)
	if err != nil {
		logger.Debugf("Failed to parse upload_certs_expiration %q: %v", value, err)
		return true
	}

	if time.Now().Add(ConfigExpirationRegeneratePeriod).After(uploadCertsExpiration) {
		logger.Debugf("Certs secret expired %s, regenerating", uploadCertsExpiration)
		return true
	}
	return false
}
