package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"k8s.io/kubernetes/cmd/kube-apiserver/app"
)

// runAPIServer will start the kubernetes api server and call the wg when it's started
// this is designed to be run in a goroutine
func runAPIServer(ctx context.Context, dataDir string, slug string) error {
	log := ctx.Value("log").(*logger.CLILogger)
	log.Info("starting kubernetes api server")

	version := "1.21.0"

	etcdUrls, etcdCtx, err := setupEtcd(ctx, dataDir, slug)
	if err != nil {
		panic(err)
	}
	go func() {
		<-etcdCtx.Done()
	}()

	serviceAccountSigningKeyFile, err := serviceAccountSigningKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "service acount signing key file")
	}
	serviceAccountKeyFile, err := serviceAccountKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "service account key file")
	}
	apiCertFile, err := apiCertFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "api cert file")
	}
	apiKeyFile, err := apiKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "api key file")
	}

	authenticationConfigFile, err := authenticationConfigFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "authentication config file")
	}

	authorizationConfigFile, err := authorizationConfigFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "authorization config file")
	}

	args := []string{
		"--service-cluster-ip-range=10.0.0.0/16",
		"--external-hostname=cluster.kots.io",
		fmt.Sprintf("--etcd-servers=%s", strings.Join(etcdUrls, ",")),
		"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
		fmt.Sprintf("--service-account-key-file=%s", serviceAccountKeyFile),
		fmt.Sprintf("--service-account-signing-key-file=%s", serviceAccountSigningKeyFile),
		"--api-audiences=https://kubernetes.default.svc.cluster.local," + version,
		// "--audit-policy-file=/auth/audit-policy.yaml",
		// "--audit-webhook-mode=batch",
		// "--audit-webhook-config-file=/auth/audit-webhook.yaml",
		"--advertise-port=443",
		"--secure-port=8443",
		"--anonymous-auth=false",
		fmt.Sprintf("--authentication-token-webhook-config-file=%s", authenticationConfigFile),
		"--authorization-mode=Node,RBAC,Webhook",
		fmt.Sprintf("--authorization-webhook-config-file=%s", authorizationConfigFile),
		"--allow-privileged=true",
		"--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota",
		fmt.Sprintf("--cert-dir=%s", filepath.Join(dataDir, "kubernetes")),
		fmt.Sprintf("--tls-cert-file=%s", apiCertFile),
		fmt.Sprintf("--tls-private-key-file=%s", apiKeyFile),
	}

	command := app.NewAPIServerCommand(ctx.Done())
	command.SetArgs(args)

	go func() {
		// TODO @divolgin: This needs better handling. Nothing will attempt to restart the server if it exits.
		logger.Infof("kubernetes api server exited %v", command.Execute())
	}()

	token := ""

	// there's a lot starting at the same time, so this needs to wait for the token to be created

	numAttempts := 0
	for token == "" && numAttempts < 10 {
		t, err := store.GetStore().GetEmbeddedClusterAuthToken()
		if store.GetStore().IsNotFound(err) {
			numAttempts++
			time.Sleep(time.Second)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "get embedded cluster auth token")
		}

		token = t
	}

	// watch the readyz endpoint to know when the api server has started
	stopWaitingAfter := time.Now().Add(time.Minute)
	for {
		url := "https://localhost:8443/readyz?verbose&exclude=etcd"

		// TODO instead of this we should pull the cert that we provisioned and use it
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := http.Client{Transport: tr}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create http request")
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(time.Second)
			continue // keep trying
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}

		if stopWaitingAfter.Before(time.Now()) {
			return errors.New("api server did not start")
		}

		time.Sleep(time.Second)
	}
}
