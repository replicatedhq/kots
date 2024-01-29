package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func main() {
	withMinio, err := strconv.ParseBool(os.Args[1])
	if err != nil {
		fmt.Println("Error parsing withMinio argument:", err)
		os.Exit(1)
	}

	gitTag := strings.Trim(os.Getenv("GIT_TAG"), "'")
	dexTag := strings.Trim(os.Getenv("DEX_TAG"), "'")
	minioTag := strings.Trim(os.Getenv("MINIO_TAG"), "'")
	rqliteTag := strings.Trim(os.Getenv("RQLITE_TAG"), "'")
	lvpTag := strings.Trim(os.Getenv("LVP_TAG"), "'")

	airgap := &kotsv1beta1.Airgap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Airgap",
		},
		Spec: kotsv1beta1.AirgapSpec{
			Format: "docker",
			SavedImages: []string{
				fmt.Sprintf("kotsadm/kotsadm:%s", gitTag),
				fmt.Sprintf("kotsadm/kotsadm-migrations:%s", gitTag),
				fmt.Sprintf("kotsadm/dex:%s", dexTag),
				fmt.Sprintf("kotsadm/rqlite:%s", rqliteTag),
				fmt.Sprintf("replicated/local-volume-provider:%s", lvpTag),
			},
		},
	}

	if withMinio {
		airgap.Spec.SavedImages = append(airgap.Spec.SavedImages, fmt.Sprintf("kotsadm/minio:%s", minioTag))
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var encoded bytes.Buffer
	if err := s.Encode(airgap, &encoded); err != nil {
		fmt.Println("Error encoding airgap file:", err)
		os.Exit(1)
	}

	err = os.WriteFile(fmt.Sprintf("%s/airgap.yaml", os.Getenv("BUNDLE_DIR")), encoded.Bytes(), 0644)
	if err != nil {
		fmt.Println("Error writing airgap.yaml:", err)
		os.Exit(1)
	}
}
