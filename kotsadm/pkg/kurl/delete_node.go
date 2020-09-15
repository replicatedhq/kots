package kurl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	configutil "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
)

func DeleteNode(ctx context.Context, client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node) error {
	err := client.CoreV1().Nodes().Delete(ctx, node.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete node")
	}

	err = purgeOSD(ctx, client, restconfig, node)
	if err != nil {
		return errors.Wrap(err, "purge OSD on node")
	}

	_, isPrimary := node.Labels["node-role.kubernetes.io/master"]
	if isPrimary {
		logger.Infof("Node %s is a primary: running etcd and kubeadm endpoints purge steps", node.Name)
		if err := purgePrimary(ctx, client, node); err != nil {
			return err
		}
	} else {
		logger.Debugf("Node %s is not a primary: skipping etcd and kubeadm endpoints purge steps", node.Name)
	}
	return nil
}

func purgeOSD(ctx context.Context, client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node) error {
	// 1. Find the Deployment for the OSD on the purged node and lookup its osd ID from labels
	// before deleting it.
	opts := metav1.ListOptions{
		LabelSelector: "app=rook-ceph-osd",
	}
	osdDeployments, err := client.AppsV1().Deployments("rook-ceph").List(ctx, opts)

	var osdID string

	for _, deployment := range osdDeployments.Items {
		hostname := deployment.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]

		if hostname == node.Name {
			osdID = deployment.Labels["ceph-osd-id"]
			logger.Infof("Deleting OSD deployment on node %s", node.Name)
			if err := client.AppsV1().Deployments("rook-ceph").Delete(ctx, deployment.Name, metav1.DeleteOptions{}); err != nil {
				return errors.Wrapf(err, "delete deployment %s", deployment.Name)
			}
			break
		}
	}

	if osdID == "" {
		logger.Infof("Failed to find ceph osd id for node %s", node.Name)
		// Not an error - rook probably isn't in use
		return nil
	}

	logger.Infof("Purging ceph OSD with id %s", osdID)

	// 2. Using the osd ID discovered in step 1, exec into the Rook operator pod and run the ceph
	// command to purge the OSD
	rookOperatorLabels := map[string]string{"app": "rook-ceph-operator"}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(rookOperatorLabels).String(),
	}
	pods, err := client.CoreV1().Pods("rook-ceph").List(ctx, listOpts)
	if err != nil {
		return errors.Wrap(err, "list Rook Operator pods")
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("Failed to purge OSD: found 0 Rook Operator pods")
	}
	exitCode, stdout, stderr, err := SyncExec(client.CoreV1(), restconfig, "rook-ceph", pods.Items[0].Name, "rook-ceph-operator", "ceph", "osd", "purge", osdID, "--yes-i-really-mean-it")
	if err != nil {
		return errors.Wrap(err, "failed to execute `ceph osd purge` in rook operator pod")
	}
	if exitCode != 0 {
		logger.Debugf("`ceph osd purge %s` stdout: %s", osdID, stdout)
		return fmt.Errorf("Failed to purge OSD: %s", stderr)
	}

	return nil
}

func purgePrimary(ctx context.Context, client kubernetes.Interface, node *corev1.Node) error {
	var remainingPrimaryIPs []string
	var purgedPrimaryIP string

	// 1. Remove the purged endpoint from kubeadm's list of API endpoints in the kubeadm-config
	// ConfigMap in the kube-system namespace. Keep the list of all primary IPs for step 2.
	clusterStatus, err := configutil.GetClusterStatus(client)
	if err != nil {
		return errors.Wrap(err, "get kube-system kubeadm-config ConfigMap ClusterStatus")
	}
	if clusterStatus.APIEndpoints == nil {
		clusterStatus.APIEndpoints = map[string]kubeadmapi.APIEndpoint{}
	}

	apiEndpoint, found := clusterStatus.APIEndpoints[node.Name]
	if found {
		purgedPrimaryIP = apiEndpoint.AdvertiseAddress
		delete(clusterStatus.APIEndpoints, node.Name)
		clusterStatusYaml, err := configutil.MarshalKubeadmConfigObject(clusterStatus)
		if err != nil {
			return errors.Wrapf(err, "marshal kubectl kubeadm cluster status without node %s", node.Name)
		}
		cm, err := client.CoreV1().ConfigMaps("kube-system").Get(ctx, kubeadmconstants.KubeadmConfigConfigMap, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "get kube-system kubeadm-config ConfigMap")
		}
		cm.Data[kubeadmconstants.ClusterStatusConfigMapKey] = string(clusterStatusYaml)
		_, err = client.CoreV1().ConfigMaps("kube-system").Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update kube-system kubeadm-config ConfigMap")
		}
		logger.Infof("Purge node %q: kubeadm-config API endpoint removed", node.Name)
	}

	for _, apiEndpoint := range clusterStatus.APIEndpoints {
		remainingPrimaryIPs = append(remainingPrimaryIPs, apiEndpoint.AdvertiseAddress)
	}

	if purgedPrimaryIP == "" {
		logger.Info("Failed to find IP of deleted primary node from kubeadm-config: skipping etcd peer removal step")
		return nil
	}
	if len(remainingPrimaryIPs) == 0 {
		return errors.New("Cannot remove etcd peer: no remaining etcd endpoints available to connect to")
	}

	// 2. Use the credentials from the mounted etcd client cert secret to connect to the remaining
	// etcd members and tell them to forget the purged member.
	removedPeerURL := "https://" + net.JoinHostPort(purgedPrimaryIP, strconv.Itoa(kubeadmconstants.EtcdListenPeerPort))
	etcdTLS, err := getEtcdTLS(ctx, "/etc/kubernetes/pki")
	if err != nil {
		return errors.Wrap(err, "get etcd certs")
	}

	var goodEtcdEndpoints []string
	for _, ip := range remainingPrimaryIPs {
		endpoint := "https://" + net.JoinHostPort(ip, strconv.Itoa(kubeadmconstants.EtcdListenClientPort))
		goodEtcdEndpoints = append(goodEtcdEndpoints, endpoint)
	}
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: goodEtcdEndpoints,
		TLS:       etcdTLS,
	})
	if err != nil {
		return errors.Wrap(err, "new etcd client")
	}
	resp, err := etcdClient.MemberList(ctx)
	if err != nil {
		return errors.Wrap(err, "list etcd members")
	}
	var purgedMemberID uint64
	for _, member := range resp.Members {
		if member.GetPeerURLs()[0] == removedPeerURL {
			purgedMemberID = member.GetID()
		}
	}
	if purgedMemberID != 0 {
		_, err = etcdClient.MemberRemove(ctx, purgedMemberID)
		if err != nil {
			return errors.Wrapf(err, "remove etcd member %d", purgedMemberID)
		}
		logger.Infof("Removed etcd member %d", purgedMemberID)
	} else {
		logger.Infof("Etcd cluster does not have member %s", removedPeerURL)
	}

	return nil
}

func getEtcdTLS(ctx context.Context, pkiDir string) (*tls.Config, error) {
	config := &tls.Config{}

	caCertPEM, err := ioutil.ReadFile(filepath.Join(pkiDir, "etcd/ca.crt"))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(caCertPEM)
	if !ok {
		return nil, errors.New("failed to append CA from pem")
	}
	config.RootCAs = pool

	clientCertPEM, err := ioutil.ReadFile(filepath.Join(pkiDir, "etcd/client.crt"))
	if err != nil {
		return nil, err
	}

	clientKeyPEM, err := ioutil.ReadFile(filepath.Join(pkiDir, "etcd/client.key"))
	if err != nil {
		return nil, err
	}

	clientCert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, err
	}
	config.Certificates = append(config.Certificates, clientCert)

	return config, nil
}
