package k8sutil

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForward(kubeContext string, localPort int, remotePort int, namespace string, podName string) (chan struct{}, error) {
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
	hostIP := strings.TrimLeft(config.Host, "htps:/")
	serverURL := url.URL{Scheme: "http", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
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
		if time.Now().Sub(start) > time.Duration(time.Second*5) {
			return nil, err
		}

		time.Sleep(time.Millisecond * 100)
	}

	return stopChan, nil
}
