package cluster

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/kubernetes/cmd/kube-apiserver/app"
)

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
		// "--authorization-mode=Node,RBAC",
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
		logger.Infof("kubernetes api server exited %v", command.Execute())
	}()

	// <-ctx.Done()

	return nil
}
