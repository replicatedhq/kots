package kotsstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (s KOTSStore) migrateSupportBundlesFromPostgres() error {
	logger.Debug("migrating support bundles from postgres")

	db := persistence.MustGetPGSession()
	query := `select id, watch_id, name, size, status, tree_index, created_at, uploaded_at, is_archived from supportbundle order by created_at desc`
	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query rows")
	}
	defer rows.Close()

	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	supportBundles := []types.SupportBundle{}
	for rows.Next() {
		var name sql.NullString
		var size sql.NullFloat64
		var treeIndex sql.NullString
		var uploadedAt sql.NullTime
		var isArchived sql.NullBool

		s := types.SupportBundle{}
		if err := rows.Scan(&s.ID, &s.AppID, &name, &size, &s.Status, &treeIndex, &s.CreatedAt, &uploadedAt, &isArchived); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		s.Name = name.String
		s.Size = size.Float64
		s.IsArchived = isArchived.Bool
		s.TreeIndex = treeIndex.String

		if uploadedAt.Valid {
			s.UploadedAt = &uploadedAt.Time
		}

		supportBundles = append(supportBundles, s)
	}

	for _, supportBundle := range supportBundles {
		analysisMarshaled := []byte{}

		// NOTE we are dropping ID, error and max_severity from the data because it's not used and has unknown validity
		query = `SELECT insights, created_at FROM supportbundle_analysis where supportbundle_id = $1`
		row := db.QueryRow(query, supportBundle.ID)
		var insightsStr sql.NullString

		a := &types.SupportBundleAnalysis{}
		hasAnalysis := true
		if err := row.Scan(&insightsStr, &a.CreatedAt); err != nil {
			if err != sql.ErrNoRows {
				return errors.Wrap(err, "failed to scan")
			}

			hasAnalysis = false
		}

		if hasAnalysis {
			if insightsStr.Valid {
				insights, err := insightsFromResults([]byte(insightsStr.String))
				if err != nil {
					return errors.Wrap(err, "failed to get insights from results")
				}

				a.Insights = insights
			}

			b, err := json.Marshal(a)
			if err != nil {
				return errors.Wrap(err, "failed to marshal analysis")
			}

			analysisMarshaled = b
		}

		query = `select redact_report from supportbundle where id = $1`
		var redactString sql.NullString
		row = db.QueryRow(query, supportBundle.ID)
		if err := row.Scan(&redactString); err != nil {
			return errors.Wrap(err, "failed to scan")
		}
		if redactString.Valid && redactString.String != "" {
			if err := s.saveSupportBundleMetafile(supportBundle.ID, "redactions", []byte(redactString.String)); err != nil {
				return errors.Wrap(err, "faile to save redactions")
			}
		}

		if err := s.saveSupportBundleMetafile(supportBundle.ID, "treeindex", []byte(supportBundle.TreeIndex)); err != nil {
			return errors.Wrap(err, "faile to save treeindex")
		}
		supportBundle.TreeIndex = ""

		bundleMarshaled, err := json.Marshal(supportBundle)
		if err != nil {
			return errors.Wrap(err, "failed to marshal bundle")
		}

		labels := kotsadmtypes.GetKotsadmLabels()
		labels["kots.io/kind"] = "supportbundle"
		labels["kots.io/appid"] = supportBundle.AppID

		secret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("supportbundle-%s", supportBundle.ID),
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels:    labels,
			},
			Data: map[string][]byte{
				"bundle":   bundleMarshaled,
				"analysis": analysisMarshaled,
			},
		}

		_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			if kuberneteserrors.IsAlreadyExists(err) {
				continue
			}
			return errors.Wrap(err, "failed to create support bundle secret")
		}
	}

	query = `delete from supportbundle`
	if _, err = db.Exec(query); err != nil {
		return errors.Wrap(err, "failed to delete support bundles from pg")
	}
	query = `delete from supportbundle_analysis`
	if _, err = db.Exec(query); err != nil {
		return errors.Wrap(err, "faild to delete support bundle analysises from pg")
	}

	return nil
}

func (s KOTSStore) ListSupportBundles(appID string) ([]*types.SupportBundle, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kots.io/kind":  "supportbundle",
			"kots.io/appid": appID,
		},
	}

	supportBundles := []*types.SupportBundle{}

	secrets, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list support bundles")
	}

	for _, secret := range secrets.Items {
		supportBundle := types.SupportBundle{}
		if err := json.Unmarshal(secret.Data["bundle"], &supportBundle); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal support bundle")
		}

		supportBundles = append(supportBundles, &supportBundle)
	}

	// sort the bundles here by date, since we don't have a sort order otherwise
	sort.Sort(types.ByCreated(supportBundles))

	return supportBundles, nil
}

func (s KOTSStore) DeletePendingSupportBundle(id string) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Delete(context.TODO(), fmt.Sprintf("pendingsupportbundle-%s", id), metav1.DeleteOptions{}); err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to delete")
	}

	return nil
}

func (s KOTSStore) ListPendingSupportBundlesForApp(appID string) ([]*types.PendingSupportBundle, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kots.io/kind":  "pendingsupportbundle",
			"kots.io/appid": appID,
		},
	}
	secrets, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})

	pendingSupportBundles := []*types.PendingSupportBundle{}

	for _, secret := range secrets.Items {
		s := types.PendingSupportBundle{}

		if err := json.Unmarshal(secret.Data["data"], &s); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal")
		}

		pendingSupportBundles = append(pendingSupportBundles, &s)
	}

	return pendingSupportBundles, nil
}

func (s KOTSStore) GetSupportBundle(id string) (*types.SupportBundle, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret")
	}

	supportBundle := types.SupportBundle{}
	if err := json.Unmarshal(secret.Data["bundle"], &supportBundle); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal")
	}

	treeindex, err := s.getSupportBundleMetafile(id, "treeindex")
	if err != nil {
		if !s.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to get treeindex from s3")
		}
	}

	supportBundle.TreeIndex = string(treeindex)

	return &supportBundle, nil
}

func (s KOTSStore) CreatePendingSupportBundle(id string, appID string, clusterID string) error {
	pendingSupportBundle := types.PendingSupportBundle{
		ID:        id,
		AppID:     appID,
		ClusterID: clusterID,
	}
	b, err := json.Marshal(pendingSupportBundle)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	labels := kotsadmtypes.GetKotsadmLabels()
	labels["kots.io/kind"] = "pendingsupportbundle"
	labels["kots.io/appid"] = appID

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pendingsupportbundle-%s", id),
			Namespace: os.Getenv("POD_NAMESPACE"),
			Labels:    labels,
		},
		Data: map[string][]byte{
			"data": b,
		},
	}

	if _, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{}); err != nil {
		return errors.Wrap(err, "failed to create secret")
	}

	return nil
}

func (s KOTSStore) CreateSupportBundle(id string, appID string, archivePath string, marshalledTree []byte) (*types.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	if err := s.saveSupportBundleMetafile(id, "treeindex", marshalledTree); err != nil {
		return nil, errors.Wrap(err, "faile to save treeindex")
	}

	// upload the bundle to s3
	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, "supportbundle.tar.gz"))

	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive file")
	}

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to upload to s3")
	}

	supportBundle := types.SupportBundle{
		ID:        id,
		Slug:      id,
		AppID:     appID,
		Size:      float64(fi.Size()),
		Status:    "uploaded",
		CreatedAt: time.Now(),
	}
	bundleMarshaled, err := json.Marshal(supportBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal support bundle")
	}

	labels := kotsadmtypes.GetKotsadmLabels()
	labels["kots.io/kind"] = "supportbundle"
	labels["kots.io/appid"] = appID

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("supportbundle-%s", id),
			Namespace: os.Getenv("POD_NAMESPACE"),
			Labels:    labels,
		},
		Data: map[string][]byte{
			"bundle":   bundleMarshaled,
			"analysis": nil,
		},
	}

	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	if _, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{}); err != nil {
		return nil, errors.Wrap(err, "failed to create secret")
	}

	return &supportBundle, nil
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func (s KOTSStore) GetSupportBundleArchive(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	newSession := awssession.New(kotss3.GetConfig())

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("supportbundles/%s/supportbundle.tar.gz", bundleID))

	outputFile, err := os.Create(filepath.Join(tmpDir, "supportbundle.tar.gz"))
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer outputFile.Close()

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(outputFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		return "", errors.Wrapf(err, "failed to download support bundle archive %q from bucket %q", *key, *bucket)
	}

	return filepath.Join(tmpDir, "supportbundle.tar.gz"), nil
}

func (s KOTSStore) GetSupportBundleAnalysis(id string) (*types.SupportBundleAnalysis, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret")
	}

	if _, ok := secret.Data["analysis"]; !ok {
		return nil, errors.New("no analysis")
	}

	if secret.Data["analysis"] == nil || len(secret.Data["analysis"]) == 0 {
		return nil, nil
	}

	a := &types.SupportBundleAnalysis{}
	if err := json.Unmarshal(secret.Data["analysis"], &a); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal analysis")
	}

	return a, nil
}

func (s KOTSStore) SetSupportBundleAnalysis(id string, results []byte) error {
	insights, err := insightsFromResults(results)
	if err != nil {
		return errors.Wrap(err, "failed to convert results to insights")
	}

	a := types.SupportBundleAnalysis{
		CreatedAt: time.Now(),
		Insights:  insights,
	}

	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list support bundle")
	}

	b, err := json.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "failed to marshal analysis")
	}

	secret.Data["analysis"] = b

	if _, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func (s KOTSStore) GetRedactions(id string) (troubleshootredact.RedactionList, error) {
	emptyRedactions := troubleshootredact.RedactionList{
		ByRedactor: map[string][]troubleshootredact.Redaction{},
		ByFile:     map[string][]troubleshootredact.Redaction{},
	}

	redactions, err := s.getSupportBundleMetafile(id, "redactions")
	if err != nil {
		return troubleshootredact.RedactionList{}, errors.Wrap(err, "failed to get redactions from s3")
	}

	if len(redactions) == 0 {
		return emptyRedactions, nil
	}

	redacts := troubleshootredact.RedactionList{}
	err = json.Unmarshal(redactions, &redacts)
	if err != nil {
		return troubleshootredact.RedactionList{}, errors.Wrap(err, "failed to unmarshal redact report")
	}

	return redacts, nil
}

func (s KOTSStore) SetRedactions(id string, redacts troubleshootredact.RedactionList) error {
	redactBytes, err := json.Marshal(redacts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal redactionlist")
	}

	if err := s.saveSupportBundleMetafile(id, "redactions", redactBytes); err != nil {
		return errors.Wrap(err, "faile to save redactions to s3")
	}

	return nil
}

func (s KOTSStore) GetSupportBundleSpecForApp(id string) (string, error) {
	q := `select supportbundle_spec from app_version
	inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
	where app.id = $1`

	spec := ""

	db := persistence.MustGetPGSession()
	row := db.QueryRow(q, id)
	err := row.Scan(&spec)
	if err != nil {
		return "", err
	}
	return spec, nil
}

func (s KOTSStore) saveSupportBundleMetafile(id string, filename string, data []byte) error {
	var gzipped bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipped)
	defer gzipWriter.Close()

	if _, err := gzipWriter.Write(data); err != nil {
		return errors.Wrap(err, "failed to write temp file")
	}
	gzipWriter.Close()

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, fmt.Sprintf("%s.gz", filename)))

	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(gzipped.Bytes()),
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return errors.Wrap(err, "failed to upload to s3")
	}

	return nil
}

func (s KOTSStore) getSupportBundleMetafile(id string, filename string) ([]byte, error) {
	newSession := awssession.New(kotss3.GetConfig())

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, fmt.Sprintf("%s.gz", filename)))

	// gzipBuffer := new(bytes.Buffer)
	// Using a temp file here because Download uses WriterAt type, which bytes.Buffer does not implement.
	gzipFile, err := ioutil.TempFile("", filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp file")
	}
	defer gzipFile.Close()
	defer os.Remove(gzipFile.Name())

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(gzipFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		if s.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to download from s3")
	}

	_, err = gzipFile.Seek(0, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to seek temp file back to 0")
	}

	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read gzip data")
	}
	defer gzipReader.Close()

	dataBuffer := new(bytes.Buffer)
	_, err = io.Copy(dataBuffer, gzipReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read gzip data")
	}

	return dataBuffer.Bytes(), nil
}

func insightsFromResults(results []byte) ([]types.SupportBundleInsight, error) {
	type Insight struct {
		Primary string `json:"primary"`
		Detail  string `json:"detail"`
	}
	type Labels struct {
		IconUri         string `json:"iconUri"`
		IconKey         string `json:"iconKey"`
		DesiredPosition string `json:"desiredPosition"`
	}
	type DBInsight struct {
		Name     string  `json:"name"`
		Severity string  `json:"severity"`
		Insight  Insight `json:"insight"`
		Labels   Labels  `json:"labels"`
	}

	dbInsights := []DBInsight{}
	if err := json.Unmarshal(results, &dbInsights); err != nil {
		logger.Error(errors.Wrap(err, "failed to unmarshal db insights"))
		dbInsights = []DBInsight{}
	}

	insights := []types.SupportBundleInsight{}
	for _, dbInsight := range dbInsights {
		desiredPosition, _ := strconv.ParseFloat(dbInsight.Labels.DesiredPosition, 64)
		insight := types.SupportBundleInsight{
			Key:             dbInsight.Name,
			Severity:        dbInsight.Severity,
			Primary:         dbInsight.Insight.Primary,
			Detail:          dbInsight.Insight.Detail,
			Icon:            dbInsight.Labels.IconUri,
			IconKey:         dbInsight.Labels.IconKey,
			DesiredPosition: desiredPosition,
		}
		insights = append(insights, insight)
	}

	return insights, nil
}
