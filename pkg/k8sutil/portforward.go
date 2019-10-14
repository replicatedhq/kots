package k8sutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func IsPortAvailable(port int) bool {
	host := ":" + strconv.Itoa(port)
	server, err := net.Listen("tcp", host)
	if err != nil {
		return false
	}

	server.Close()
	return true
}

func PortForward(kubeContext string, localPort int, remotePort int, namespace string, podName string, pollForAdditionalPorts bool) (chan struct{}, error) {
	if !IsPortAvailable(localPort) {
		return nil, errors.Errorf("Unable to connect to cluster. There's another process using port %d.", localPort)
	}

	// port forward
	config, err := clientcmd.BuildConfigFromFlags("", kubeContext)
	if err != nil {
		return nil, err
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	scheme := ""
	hostIP := config.Host

	u, err := url.Parse(config.Host)
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

	// Block until the new service is responding, limited to (math) seconds
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		response, err := quickClient.Get(fmt.Sprintf("http://localhost:%d", localPort))
		if err == nil && response.StatusCode == http.StatusOK {
			break
		}
		if time.Now().Sub(start) > time.Duration(time.Second*10) {
			return nil, err
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

		uri := fmt.Sprintf("http://localhost:%d/api/v1/kots/ports", localPort)
		go func() error {
			for {
				req, err := http.NewRequest("GET", uri, nil)
				if err != nil {
					return errors.Wrap(err, "failed to create request")
				}
				req.Header.Set("Accept", "application/json")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return errors.Wrap(err, "failed to get ports")
				}

				defer resp.Body.Close()
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return errors.Wrap(err, "failed to parse response")
				}

				desiredAdditionalPorts := []AdditionalPort{}
				if err := json.Unmarshal(b, &desiredAdditionalPorts); err != nil {
					return errors.Wrap(err, "failed to decode response")
				}

				for _, desiredAdditionalPort := range desiredAdditionalPorts {
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

					serviceStopCh, err := ServiceForward(kubeContext, desiredAdditionalPort.LocalPort, desiredAdditionalPort.ServicePort, namespace, desiredAdditionalPort.ServiceName)
					if err != nil {
						continue // try again
					}
					if serviceStopCh == nil {
						// we didn't do the port forwarding, probably because the pod isn't ready.
						// try again next loop
						continue // try again
					}

					forwardedAdditionalPorts[desiredAdditionalPort] = serviceStopCh
					log := logger.NewLogger()
					log.Info("Go to http://localhost:%d to access the application", desiredAdditionalPort.LocalPort)
				}
				time.Sleep(time.Second * 5)
			}
		}()
	}

	return stopChan, nil
}

func ServiceForward(kubeContext string, localPort int, remotePort int, namespace string, serviceName string) (chan struct{}, error) {
	if !IsPortAvailable(localPort) {
		return nil, errors.Errorf("Unable to connect to cluster. There's another process using port %d.", localPort)
	}

	// port forward
	config, err := clientcmd.BuildConfigFromFlags("", kubeContext)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(serviceName, metav1.GetOptions{})
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

	roundTripper, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	scheme := ""
	hostIP := config.Host

	u, err := url.Parse(config.Host)
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

	podList, err := clientset.CoreV1().Pods(namespace).List(options)
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
