package k8sutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func IsPortAvailable(port int) bool {
	// portforward explicitly listens on localhost
	host := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	server, err := net.Listen("tcp4", host)
	if err != nil {
		return false
	}

	server.Close()
	return true
}

// PortForward starts a local port forward to a pod in the cluster
// if localport is set, it will attempt to use that port locally.
// always check the port number returned though, because a port conflict
// could cause a different port to be used
func PortForward(localPort int, remotePort int, namespace string, podName string, pollForAdditionalPorts bool, stopCh <-chan struct{}, log *logger.CLILogger) (int, <-chan error, error) {
	if localPort == 0 {
		freePort, err := freeport.GetFreePort()
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to get free port")
		}

		localPort = freePort
	}

	if !IsPortAvailable(localPort) {
		freePort, err := freeport.GetFreePort()
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to get free port")
		}

		localPort = freePort
	}

	// port forward
	cfg, err := GetClusterConfig()
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get cluster config")
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to create roundtriper")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	scheme := ""
	hostIP := cfg.Host

	u, err := url.Parse(cfg.Host)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to parse host")
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
		// Locks until stopChan is closed.
		// The main function may timeout before this returns an error
		forwardErr = forwarder.ForwardPorts()
		if forwardErr != nil {
			errChan <- errors.Wrap(forwardErr, "forward ports")
		}
	}()

	// Block until the new service is responding, limited to (math) seconds
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		if forwardErr != nil {
			return 0, nil, forwardErr
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

		// This process is long lived, avoid creating too many clientsets here
		// https://github.com/kubernetes/client-go/issues/803
		clientset, err := GetClientset()
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to get clientset")
		}

		consecutiveErrorsLogged := struct {
			read      int
			unmarshal int
		}{
			read:      0,
			unmarshal: 0,
		}

		sleepTime := time.Second
		go func() {
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
				b, err := ioutil.ReadAll(resp.Body)
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
						log.Error(errors.Wrap(err, fmt.Sprintf("failed to execute kubectl port-forward -n %s svc/%s %d:%d", namespace, desiredAdditionalPort.ServiceName, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort)))
						continue // try again
					}
					if serviceStopCh == nil {
						// we didn't do the port forwarding, probably because the pod isn't ready.
						// try again next loop
						// The API doesn't return ports that aren't ready, so this is possibly rbac?
						log.Error(errors.Errorf("failed to forward port. check that you have permission to run kubectl port-forward -n %s svc/%s %d:%d", namespace, desiredAdditionalPort.ServiceName, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort))
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

func ServiceForward(clientset *kubernetes.Clientset, cfg *rest.Config, localPort int, remotePort int, namespace string, serviceName string) (chan struct{}, error) {
	if !IsPortAvailable(localPort) {
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
