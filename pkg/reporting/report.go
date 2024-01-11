package reporting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ReportSecretNameFormat = "kotsadm-%s-report"
	ReportSecretKey        = "report"
	ReportEventLimit       = 4000
	ReportSizeLimit        = 1 * 1024 * 1024 // 1MB
)

type ReportType string

const (
	ReportTypeInstance  ReportType = "instance"
	ReportTypePreflight ReportType = "preflight"
)

type Report interface {
	GetType() ReportType
	GetSecretName(appSlug string) string
	GetSecretKey() string
	AppendEvents(report Report) error
	GetEventLimit() int
	GetMtx() *sync.Mutex
}

var _ Report = &InstanceReport{}
var _ Report = &PreflightReport{}

func AppendReport(clientset kubernetes.Interface, namespace string, appSlug string, report Report) error {
	report.GetMtx().Lock()
	defer report.GetMtx().Unlock()

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), report.GetSecretName(appSlug), metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get report secret")
	} else if kuberneteserrors.IsNotFound(err) {
		data, err := EncodeReport(report)
		if err != nil {
			return errors.Wrap(err, "failed to encode report")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      report.GetSecretName(appSlug),
				Namespace: namespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				report.GetSecretKey(): data,
			},
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create report secret")
		}

		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	var existingReport Report
	if existingSecret.Data[report.GetSecretKey()] != nil {
		existingReport, err = DecodeReport(existingSecret.Data[report.GetSecretKey()], report.GetType())
		if err != nil {
			return errors.Wrap(err, "failed to load existing report")
		}

		if err := existingReport.AppendEvents(report); err != nil {
			return errors.Wrap(err, "failed to append events to existing report")
		}
	} else {
		// secret exists but doesn't have the report key, so just use the report that was passed in
		existingReport = report
	}

	data, err := EncodeReport(existingReport)
	if err != nil {
		return errors.Wrap(err, "failed to encode existing report")
	}

	existingSecret.Data[report.GetSecretKey()] = data

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update report secret")
	}

	return nil
}

func EncodeReport(r Report) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal report")
	}
	compressedData, err := util.GzipData(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to gzip report")
	}
	encodedData := base64.StdEncoding.EncodeToString(compressedData)

	return []byte(encodedData), nil
}

func DecodeReport(encodedData []byte, reportType ReportType) (Report, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode report")
	}
	decompressedData, err := util.GunzipData(decodedData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to gunzip report")
	}

	var r Report
	switch reportType {
	case ReportTypeInstance:
		r = &InstanceReport{}
		if err := json.Unmarshal(decompressedData, r); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal instance report")
		}
	case ReportTypePreflight:
		r = &PreflightReport{}
		if err := json.Unmarshal(decompressedData, r); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal preflight report")
		}
	default:
		return nil, errors.Errorf("unknown report type %q", reportType)
	}

	return r, nil
}
