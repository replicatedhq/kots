package k8sutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func IsPortAvailable(clientset kubernetes.Interface, port int) (bool, error) {
	// kube-proxy stopped opening ports on the host for NodePort services in recent versions of kubernetes,
	// so net.Listen won't be able to detect if the port is used by a NodePort service or not, and port forwarding might hang or redirect to the wrong component.
	// https://github.com/kubernetes/kubernetes/pull/108496
	// so we check if the port is used by a NodePort service.
	services, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		// TODO: make this work with minimal RBAC
		if !kuberneteserrors.IsForbidden(err) {
			return false, errors.Wrap(err, "failed to list services")
		}
	}

	for _, service := range services.Items {
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			continue
		}
		for _, p := range service.Spec.Ports {
			if p.NodePort > 0 && int(p.NodePort) == port {
				return false, nil
			}
		}
	}

	// portforward explicitly listens on localhost
	host := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	server, err := net.Listen("tcp4", host)
	if err != nil {
		return false, nil
	}

	server.Close()
	return true, nil
}

func FindFreePort(clientset kubernetes.Interface) (int, error) {
	return findFreePort(clientset, 10, 1)
}

func findFreePort(clientset kubernetes.Interface, maxAttempts int, currAttempt int) (int, error) {
	if currAttempt > maxAttempts {
		return 0, errors.New(fmt.Sprintf("Timed out after %d attempts", maxAttempts))
	}

	port, err := freeport.GetFreePort()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get free port")
	}

	isPortAvailable, err := IsPortAvailable(clientset, port)
	if err != nil {
		return 0, errors.Wrap(err, "failed to check if port is available")
	}

	if !isPortAvailable {
		return findFreePort(clientset, maxAttempts, currAttempt+1)
	}

	return port, nil
}

// PortForward starts a local port forward to a pod in the cluster
// if localport is set, it will attempt to use that port locally.
// always check the port number returned though, because a port conflict
// could cause a different port to be used
func PortForward(localPort int, remotePort int, namespace string, getPodName func() (string, error), pollForAdditionalPorts bool, stopCh <-chan struct{}, log *logger.CLILogger) (int, chan error, error) {
	// This process is long lived, avoid creating too many clientsets here
	// https://github.com/kubernetes/client-go/issues/803
	clientset, err := GetClientset()
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get clientset")
	}

	if localPort == 0 {
		freePort, err := FindFreePort(clientset)
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to find free port")
		}
		localPort = freePort
	} else {
		isPortAvailable, err := IsPortAvailable(clientset, localPort)
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to check if port is available")
		}
		if !isPortAvailable {
			freePort, err := FindFreePort(clientset)
			if err != nil {
				return 0, nil, errors.Wrap(err, "failed to find free port")
			}

			localPort = freePort
		}
	}

	podName, err := getPodName()
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get pod name for port forward")
	}

	// port forward
	cfg, err := GetClusterConfig()
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get cluster config")
	}

	dialer, err := createDialer(cfg, namespace, podName)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to create dialer")
	}

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to create new portforward")
	}

	errChan := make(chan error, 2) // 2 go routines are writing to this channel

	go func() {
		for range readyChan {
		}

		if len(errOut.String()) != 0 {
			errChan <- errors.Errorf("remote error: %s", errOut)
		} else if len(out.String()) != 0 {
			// fmt.Println(out.String())
		}
	}()

	var forwardErr error
	go func() {
		for {
			// Locks until stopChan is closed or connection to pod is lost.
			// The main function may timeout before this returns an error
			forwardErr = forwarder.ForwardPorts()
			if forwardErr != nil {
				errChan <- errors.Wrap(forwardErr, "forward ports")
			} else {
				// Connection to pod was lost or stopChan was closed
				log.Info("lost connection to pod %s", podName)
				success := false
				delaySeconds := time.Duration(0)
				for !success {
					delaySeconds += 1
					if delaySeconds > 5 {
						delaySeconds = 5
					}
					time.Sleep(delaySeconds * time.Second)
					log.Info("attempting to re-establish port-forward")
					podName, err = getPodName()
					if err != nil {
						log.Error(errors.Wrap(err, "failed to get pod name"))
						continue
					}
					dialer, err = createDialer(cfg, namespace, podName)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to create dialer"))
						continue
					}
					stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
					out, errOut := new(bytes.Buffer), new(bytes.Buffer)
					forwarder, err = portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, out, errOut)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to create new portforward"))
						continue
					}
					success = true
					log.Info("re-established port-forward to %s", podName)
				}
			}
		}
	}()

	// Block until the new service is responding, limited to (math) seconds
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		if forwardErr != nil {
			return 0, nil, errors.Wrap(forwardErr, "failed to forward ports")
		}

		response, err := quickClient.Get(fmt.Sprintf("http://localhost:%d/healthz", localPort))
		if err == nil && response.StatusCode == http.StatusOK {
			break
		}
		if time.Now().Sub(start) > time.Duration(time.Second*10) {
			if err == nil {
				err = errors.Errorf("service responded with status %s", response.Status)
			}
			return 0, nil, errors.Wrap(err, "failed to query healthz")
		}

		time.Sleep(time.Millisecond * 100)
		if quickClient.Timeout < time.Second {
			quickClient.Timeout = quickClient.Timeout + time.Millisecond*100
		}
	}

	if pollForAdditionalPorts {
		type AdditionalPort struct {
			ServiceName string `json:"serviceName"`
			ServicePort int    `json:"servicePort"`
			LocalPort   int    `json:"localPort"`
		}

		forwardedAdditionalPorts := map[AdditionalPort]chan struct{}{}

		keepPolling := true
		go func() {
			<-stopChan
			keepPolling = false
		}()

		uri := fmt.Sprintf("http://localhost:%d/api/v1/kots/ports", localPort)

		consecutiveErrorsLogged := struct {
			read      int
			unmarshal int
		}{
			read:      0,
			unmarshal: 0,
		}

		sleepTime := time.Second
		go func() {
			prevServiceForwardErrMap := map[string]error{}
			for keepPolling {
				time.Sleep(sleepTime)
				sleepTime = time.Second * 5

				req, err := http.NewRequest("GET", uri, nil)
				if err != nil {
					log.Error(errors.Wrap(err, "failed to create request"))
					continue
				}
				req.Header.Set("Accept", "application/json")

				authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
				if err != nil {
					log.Error(errors.Wrap(err, "failed to get kotsadm auth slug"))
					continue
				}
				req.Header.Add("Authorization", authSlug)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Info("failed execute http request while listing additional ports: %v", err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					// Don't log server side errors.  This will happen when app has not been installed yet,
					// for example, and relevant logs should be in the kotsadm pod.
					continue
				}

				defer resp.Body.Close()
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					if consecutiveErrorsLogged.read == 0 {
						log.Info("failed to read response body while listing additional ports: %v", err)
						consecutiveErrorsLogged.read++
					}
					continue
				}
				consecutiveErrorsLogged.read = 0

				response := struct {
					DesiredAdditionalPorts []AdditionalPort `json:"ports"`
				}{}
				if err := json.Unmarshal(b, &response); err != nil {
					if consecutiveErrorsLogged.unmarshal == 0 {
						log.Info("failed to decode response while listing additional ports: %s", b)
						consecutiveErrorsLogged.unmarshal++
					}
					continue
				}
				consecutiveErrorsLogged.unmarshal = 0

				for _, desiredAdditionalPort := range response.DesiredAdditionalPorts {
					alreadyForwarded := false
					for forwardedAdditionalPort := range forwardedAdditionalPorts {
						// Avoid port conflicts by only taking the first to claim a local port
						if forwardedAdditionalPort.LocalPort == desiredAdditionalPort.LocalPort {
							alreadyForwarded = true
							break
						}
					}

					if alreadyForwarded {
						continue
					}

					serviceStopCh, err := ServiceForward(clientset, cfg, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort, namespace, desiredAdditionalPort.ServiceName)
					if err != nil {
						prevServiceForwardErr := prevServiceForwardErrMap[desiredAdditionalPort.ServiceName]
						if prevServiceForwardErr == nil || prevServiceForwardErr.Error() != err.Error() {
							log.Error(errors.Wrap(err, fmt.Sprintf("failed to execute kubectl port-forward -n %s svc/%s %d:%d", namespace, desiredAdditionalPort.ServiceName, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort)))
							prevServiceForwardErrMap[desiredAdditionalPort.ServiceName] = err
						}
						continue // try again
					}
					if serviceStopCh == nil {
						// we didn't do the port forwarding, probably because the pod isn't ready.
						// try again next loop
						// The API doesn't return ports that aren't ready, so this is possibly rbac?
						err = errors.Errorf("failed to forward port. check that you have permission to run kubectl port-forward -n %s svc/%s %d:%d", namespace, desiredAdditionalPort.ServiceName, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort)
						prevServiceForwardErr := prevServiceForwardErrMap[desiredAdditionalPort.ServiceName]
						if prevServiceForwardErr == nil || prevServiceForwardErr.Error() != err.Error() {
							log.Error(err)
							prevServiceForwardErrMap[desiredAdditionalPort.ServiceName] = err
						}
						continue // try again
					}

					forwardedAdditionalPorts[desiredAdditionalPort] = serviceStopCh
					log.ActionWithoutSpinner("Go to http://localhost:%d to access the application", desiredAdditionalPort.LocalPort)
				}
			}
		}()
	}

	return localPort, errChan, nil
}

func createDialer(cfg *rest.Config, namespace string, podName string) (httpstream.Dialer, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create roundtriper")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	scheme := ""
	hostIP := cfg.Host

	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse host")
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		scheme = u.Scheme
		hostIP = u.Host
	}

	serverURL := url.URL{Scheme: scheme, Path: path, Host: hostIP}
	return spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL), nil
}

func ServiceForward(clientset *kubernetes.Clientset, cfg *rest.Config, localPort int, remotePort int, namespace string, serviceName string) (chan struct{}, error) {
	isPortAvailable, err := IsPortAvailable(clientset, localPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if port is available")
	}
	if !isPortAvailable {
		return nil, errors.Errorf("Unable to connect to cluster. There's another process using port %d.", localPort)
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service")
	}

	selector := labels.SelectorFromSet(svc.Spec.Selector)
	podName, err := getFirstPod(clientset, namespace, selector.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get first pod")
	}

	if podName == "" {
		// not ready yet
		return nil, nil
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	scheme := ""
	hostIP := cfg.Host

	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		scheme = u.Scheme
		hostIP = u.Host
	}

	serverURL := url.URL{Scheme: scheme, Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		for range readyChan {
		}

		if len(errOut.String()) != 0 {
			panic(errOut.String())
		} else if len(out.String()) != 0 {
			// fmt.Println(out.String())
		}
	}()

	go func() error {
		if err = forwarder.ForwardPorts(); err != nil { // Locks until stopChan is closed.
			panic(err)
		}

		return nil
	}()

	return stopChan, nil
}

func getFirstPod(clientset *kubernetes.Clientset, namespace string, selector string) (string, error) {
	options := metav1.ListOptions{LabelSelector: selector}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), options)
	if err != nil {
		return "", errors.Wrap(err, "failed to list pods")
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			isNotReady := false
			for _, status := range pod.Status.ContainerStatuses {
				if !status.Ready {
					isNotReady = true
				}
			}

			if !isNotReady {
				return pod.Name, nil
			}
		}
	}

	return "", nil
}
