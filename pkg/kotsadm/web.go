package kotsadm

import (
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	executableMode = int32(511) // hex 777
)

func getWebYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var config bytes.Buffer
	if err := s.Encode(webConfig(deployOptions), &config); err != nil {
		return nil, errors.Wrap(err, "failed to marshal web config")
	}
	docs["web-config.yaml"] = config.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(webDeployment(deployOptions), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marsha web deployment")
	}
	docs["web-deployment.yaml"] = deployment.Bytes()

	var service bytes.Buffer
	if err := s.Encode(webService(deployOptions), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal web service")
	}
	docs["web-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureWeb(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	if deployOptions.Hostname == "" {
		hostname, err := promptForHostname()
		if err != nil {
			return errors.Wrap(err, "failed to prompt for hostname")
		}

		deployOptions.Hostname = hostname
	}

	if deployOptions.ServiceType == "" {
		serviceType, err := promptForWebServiceType(deployOptions)
		if err != nil {
			return errors.Wrap(err, "failed to prompt for service type")
		}

		deployOptions.ServiceType = serviceType
	}

	if err := ensureWebConfig(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web configmap")
	}

	if err := ensureWebDeployment(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web deployment")
	}

	if err := ensureWebService(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web service")
	}

	return nil
}

func ensureWebConfig(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get("kotsadm-web-scripts", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing config map")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(webConfig(*deployOptions))
		if err != nil {
			return errors.Wrap(err, "failed to create configmap")
		}
	}

	return nil
}

func ensureWebDeployment(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-web", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(webDeployment(deployOptions))
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureWebService(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(deployOptions.Namespace).Get("kotsadm-web", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(deployOptions.Namespace).Create(webService(*deployOptions))
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}

func promptForWebServiceType(deployOptions *types.DeployOptions) (string, error) {
	prompt := promptui.Select{
		Label: "Web/UI Service Type:",
		Items: []string{"ClusterIP", "NodePort", "LoadBalancer"},
	}

	for {
		_, result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		if result == "NodePort" {
			nodePort, err := promptForWebNodePort()
			if err != nil {
				return "", errors.Wrap(err, "failed to prompt for node port")
			}

			deployOptions.NodePort = int32(nodePort)
		}
		return result, nil
	}

}

func promptForWebNodePort() (int, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Node Port:",
		Templates: templates,
		Default:   "30000",
		Validate: func(input string) error {
			_, err := strconv.Atoi(input)
			return err
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

		nodePort, err := strconv.Atoi(result)
		if err != nil {
			return 0, errors.Wrap(err, "failed to convert nodeport")
		}

		return nodePort, nil
	}

}

func promptForHostname() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Hostname for the Admin Console:",
		Templates: templates,
		Default:   "localhost:8800",
		Validate: func(input string) error {
			if !strings.Contains(input, ":") {
				errs := validation.IsDNS1123Subdomain(input)
				if len(errs) > 0 {
					return errors.New(errs[0])
				}

				return nil
			}

			split := strings.Split(input, ":")
			if len(split) != 2 {
				return errors.New("only hostname or hostname:port are allowed formats")
			}

			errs := validation.IsDNS1123Subdomain(split[0])
			if len(errs) > 0 {
				return errors.New(errs[0])
			}

			_, err := strconv.Atoi(split[1])
			if err != nil {
				return err
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
