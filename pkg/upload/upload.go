package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type UploadOptions struct {
	Namespace       string
	Kubeconfig      string
	ExistingAppSlug string
	NewAppName      string
}

func Upload(path string, uploadOptions UploadOptions) error {
	archiveFilename, err := createUploadableArchive(path)
	if err != nil {
		return errors.Wrap(err, "failed to create uploadable archive")
	}
	defer os.Remove(archiveFilename)

	// Make sure we have a name or slug
	if uploadOptions.ExistingAppSlug == "" && uploadOptions.NewAppName == "" {
		split := strings.Split(path, string(os.PathSeparator))
		lastPathPart := ""
		idx := 1
		for lastPathPart == "" {
			lastPathPart = split[len(split)-idx]
			if lastPathPart == "" && len(split) > idx {
				idx++
				continue
			}

			break
		}

		appName, err := relentlesslyPromptForAppName(lastPathPart)
		if err != nil {
			return errors.Wrap(err, "failed to prompt for app name")
		}

		uploadOptions.NewAppName = appName
	}

	// Find the kotadm-api pod
	podName, err := findKotsadm(uploadOptions)
	if err != nil {
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	// set up port forwarding to get to it
	stopCh, err := k8sutil.PortForward(uploadOptions.Kubeconfig, 3000, 3000, uploadOptions.Namespace, podName)
	if err != nil {
		return errors.Wrap(err, "failed to start port forwarding")
	}
	defer close(stopCh)

	// upload using http to the pod directly
	req, err := createUploadRequest(archiveFilename, uploadOptions.ExistingAppSlug, uploadOptions.NewAppName, "http://localhost:3000/api/v1/kots")
	if err != nil {
		time.Sleep(time.Minute * 5)
		return errors.Wrap(err, "failed to upload")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}

	if resp.StatusCode != 200 {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}
	type UploadResponse struct {
		URI string `json:"uri"`
	}
	var uploadResponse UploadResponse
	if err := json.Unmarshal(b, &uploadResponse); err != nil {
		return errors.Wrap(err, "failed to unmarshal response")
	}

	return nil
}

func findKotsadm(uploadOptions UploadOptions) (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes clientset")
	}

	pods, err := clientset.CoreV1().Pods(uploadOptions.Namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-api"})
	if err != nil {
		return "", errors.Wrap(err, "failed to list pods")
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", errors.New("unable to find kotsadm pod")
}

func createUploadRequest(path string, existingAppSlug string, newAppName string, uri string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	archivePart, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create form file")
	}
	_, err = io.Copy(archivePart, file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to copy file to upload")
	}

	method := ""
	if existingAppSlug != "" {
		method = "PUT"
		metadata := map[string]string{
			"slug": existingAppSlug,
		}
		b, err := json.Marshal(metadata)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal json")
		}
		metadataPart, err := writer.CreateFormField("metadata")
		if err != nil {
			return nil, errors.Wrap(err, "failed to add metadata")
		}
		if _, err := io.Copy(metadataPart, bytes.NewReader(b)); err != nil {
			return nil, errors.Wrap(err, "failed to copy metadata")
		}
	} else {
		method = "POST"
		metadata := map[string]string{
			"name": newAppName,
		}
		b, err := json.Marshal(metadata)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal json")
		}
		metadataPart, err := writer.CreateFormField("metadata")
		if err != nil {
			return nil, errors.Wrap(err, "failed to add metadata")
		}
		if _, err := io.Copy(metadataPart, bytes.NewReader(b)); err != nil {
			return nil, errors.Wrap(err, "failed to copy metadata")
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to close writer")
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new request")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func relentlesslyPromptForAppName(defaultAppName string) (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Application name:",
		Templates: templates,
		Default:   defaultAppName,
		Validate: func(input string) error {
			if len(input) < 3 {
				return errors.New("invalid app name")
			}
			return nil
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return result, nil
	}

}
