package kotsstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
	"github.com/rqlite/gorqlite"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (s *KOTSStore) migrateSupportBundlesFromRqlite() error {
	logger.Debug("migrating support bundles from rqlite")

	db := persistence.MustGetDBSession()
	query := `select id, watch_id, name, size, status, tree_index, created_at, uploaded_at, shared_at, is_archived from supportbundle order by created_at desc`
	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	supportBundles := []types.SupportBundle{}
	for rows.Next() {
		var name gorqlite.NullString
		var size gorqlite.NullFloat64
		var treeIndex gorqlite.NullString
		var uploadedAt gorqlite.NullTime
		var sharedAt gorqlite.NullTime
		var isArchived gorqlite.NullBool

		s := types.SupportBundle{}
		if err := rows.Scan(&s.ID, &s.AppID, &name, &size, &s.Status, &treeIndex, &s.CreatedAt, &uploadedAt, &sharedAt, &isArchived); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		s.Slug = s.ID
		s.Name = name.String
		s.Size = size.Float64
		s.IsArchived = isArchived.Bool
		s.TreeIndex = treeIndex.String

		if uploadedAt.Valid {
			s.UploadedAt = &uploadedAt.Time
		}

		if sharedAt.Valid {
			s.SharedAt = &sharedAt.Time
		}

		supportBundles = append(supportBundles, s)
	}

	for _, supportBundle := range supportBundles {
		analysisMarshaled := []byte{}

		// NOTE we are dropping ID, error and max_severity from the data because it's not used and has unknown validity
		query = `SELECT insights, created_at FROM supportbundle_analysis where supportbundle_id = ?`
		rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{supportBundle.ID},
		})
		if err != nil {
			return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
		}
		if !rows.Next() {
			return ErrNotFound
		}

		var insightsStr gorqlite.NullString
		a := &types.SupportBundleAnalysis{}
		hasAnalysis := true
		if err := rows.Scan(&insightsStr, &a.CreatedAt); err != nil {
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

		query = `select redact_report from supportbundle where id = ?`
		rows, err = db.QueryOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{supportBundle.ID},
		})
		if err != nil {
			return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
		}
		if !rows.Next() {
			return ErrNotFound
		}

		var redactString gorqlite.NullString
		if err := rows.Scan(&redactString); err != nil {
			return errors.Wrap(err, "failed to scan")
		}
		if redactString.Valid && redactString.String != "" {
			if err := s.saveSupportBundleMetafile(supportBundle.ID, "redactions", []byte(redactString.String)); err != nil {
				return errors.Wrap(err, "faile to save redactions")
			}
		}

		if err := s.saveSupportBundleMetafile(supportBundle.ID, "treeindex", []byte(supportBundle.TreeIndex)); err != nil {
			return errors.Wrap(err, "failed to save treeindex")
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
				Namespace: util.PodNamespace,
				Labels:    labels,
			},
			Data: map[string][]byte{
				"bundle":   bundleMarshaled,
				"analysis": analysisMarshaled,
			},
		}

		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			if kuberneteserrors.IsAlreadyExists(err) {
				continue
			}
			return errors.Wrap(err, "failed to create support bundle secret")
		}
	}

	query = `delete from supportbundle`
	if wr, err := db.WriteOne(query); err != nil {
		return fmt.Errorf("failed to delete support bundles from db: %v: %v", err, wr.Err)
	}
	query = `delete from supportbundle_analysis`
	if wr, err := db.WriteOne(query); err != nil {
		return fmt.Errorf("failed to delete support bundle analysises from db: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) ListSupportBundles(appID string) ([]*types.SupportBundle, error) {
	supportBundles := []*types.SupportBundle{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kots.io/kind":  "supportbundle",
			"kots.io/appid": appID,
		},
	}

	secrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list support bundles")
	}

	for _, secret := range secrets.Items {
		supportBundle, err := s.getSupportBundleFromSecretData(&secret)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal support bundle")
		}
		supportBundles = append(supportBundles, supportBundle)
	}

	// sort the bundles here by date, since we don't have a sort order otherwise
	sort.Sort(sort.Reverse(types.ByCreated(supportBundles)))

	return supportBundles, nil
}

func (s *KOTSStore) GetSupportBundle(id string) (*types.SupportBundle, error) {
	supportBundle := &types.SupportBundle{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret")
	}

	supportBundle, err = s.getSupportBundleFromSecretData(secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal")
	}

	if supportBundle != nil {
		treeindex, err := s.getSupportBundleMetafile(id, "treeindex")
		if err != nil {
			if s.IsNotFound(err) {
				return supportBundle, nil
			}
			return nil, errors.Wrap(err, "failed to get treeindex")
		}
		supportBundle.TreeIndex = string(treeindex)
	}

	return supportBundle, nil
}

func (s *KOTSStore) DeleteSupportBundle(bundleID string, appID string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	// delete the secret
	if err := clientset.CoreV1().Secrets(util.PodNamespace).Delete(context.TODO(), fmt.Sprintf("supportbundle-%s", bundleID), metav1.DeleteOptions{}); err != nil && !s.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete secret")
	}

	// delete the archive
	sbPath := filepath.Join("supportbundles", bundleID)
	if err := filestore.GetStore().DeleteArchive(sbPath); err != nil {
		return errors.Wrap(err, "failed to delete archive")
	}
	return nil
}

func (s *KOTSStore) CreateInProgressSupportBundle(supportBundle *types.SupportBundle) error {
	id := supportBundle.ID
	appID := supportBundle.AppID
	now := time.Now()

	supportBundle.Status = types.BUNDLE_RUNNING
	supportBundle.CreatedAt = now
	supportBundle.UpdatedAt = &now

	bundleMarshaled, err := json.Marshal(supportBundle)
	if err != nil {
		return errors.Wrap(err, "failed to marshal support bundle")
	}

	labels := kotsadmtypes.GetKotsadmLabels()
	labels["kots.io/kind"] = "supportbundle"
	labels["kots.io/appid"] = appID
	labels["kots.io/status"] = string(types.BUNDLE_RUNNING)

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("supportbundle-%s", id),
			Namespace: util.PodNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"bundle":   bundleMarshaled,
			"analysis": nil,
		},
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if _, err := clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), &secret, metav1.CreateOptions{}); err != nil {
		return errors.Wrap(err, "failed to create secret")
	}

	return nil
}

func (s *KOTSStore) CreateSupportBundle(id string, appID string, archivePath string, marshalledTree []byte) (*types.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	if err := s.saveSupportBundleMetafile(id, "treeindex", marshalledTree); err != nil {
		return nil, errors.Wrap(err, "failed to save treeindex")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive file")
	}
	defer f.Close()

	outputPath := filepath.Join("supportbundles", id, "supportbundle.tar.gz")
	err = filestore.GetStore().WriteArchive(outputPath, f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write archive")
	}

	now := time.Now()
	supportBundle := types.SupportBundle{
		ID:        id,
		Slug:      id,
		AppID:     appID,
		Size:      float64(fi.Size()),
		Status:    types.BUNDLE_UPLOADED,
		CreatedAt: now,
		UpdatedAt: &now,
	}

	bundleMarshaled, err := json.Marshal(supportBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal support bundle")
	}

	labels := kotsadmtypes.GetKotsadmLabels()
	labels["kots.io/kind"] = "supportbundle"
	labels["kots.io/appid"] = appID
	labels["kots.io/status"] = string(types.BUNDLE_UPLOADED)

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("supportbundle-%s", id),
			Namespace: util.PodNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"bundle":   bundleMarshaled,
			"analysis": nil,
		},
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	if _, err := clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), &secret, metav1.CreateOptions{}); err != nil {
		return nil, errors.Wrap(err, "failed to create secret")
	}

	return &supportBundle, nil
}

// UpdateSupportBundle updates the support bundle definition in the secret
func (s *KOTSStore) UpdateSupportBundle(bundle *types.SupportBundle) error {
	now := time.Now()
	bundle.UpdatedAt = &now

	supportBundleSecretMtx.Lock()
	defer supportBundleSecretMtx.Unlock()

	marshaledBundle, err := json.Marshal(bundle)
	if err != nil {
		return errors.Wrap(err, "failed to marshal support bundle")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", bundle.ID), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list support bundle")
	}

	secret.ObjectMeta.Labels["kots.io/status"] = string(bundle.Status)

	secret.Data["bundle"] = marshaledBundle

	if _, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

// UploadSupportBundle pushes the metadata file and support bundle archive to the file store
func (s *KOTSStore) UploadSupportBundle(id string, archivePath string, marshalledTree []byte) error {

	if err := s.saveSupportBundleMetafile(id, "treeindex", marshalledTree); err != nil {
		return errors.Wrap(err, "failed to save treeindex")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive file")
	}
	defer f.Close()

	outputPath := filepath.Join("supportbundles", id, "supportbundle.tar.gz")
	err = filestore.GetStore().WriteArchive(outputPath, f)
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	return nil
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func (s *KOTSStore) GetSupportBundleArchive(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	path := fmt.Sprintf("supportbundles/%s/supportbundle.tar.gz", bundleID)
	archivePath, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to read archive")
	}

	return archivePath, nil
}

func (s *KOTSStore) GetSupportBundleAnalysis(id string) (*types.SupportBundleAnalysis, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
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

func (s *KOTSStore) SetSupportBundleAnalysis(id string, results []byte) error {
	insights, err := insightsFromResults(results)
	if err != nil {
		return errors.Wrap(err, "failed to convert results to insights")
	}

	a := types.SupportBundleAnalysis{
		CreatedAt: time.Now(),
		Insights:  insights,
	}

	supportBundleSecretMtx.Lock()
	defer supportBundleSecretMtx.Unlock()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), fmt.Sprintf("supportbundle-%s", id), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list support bundle")
	}

	b, err := json.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "failed to marshal analysis")
	}

	secret.Data["analysis"] = b

	if _, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func (s *KOTSStore) GetRedactions(id string) (troubleshootredact.RedactionList, error) {
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

func (s *KOTSStore) SetRedactions(id string, redacts troubleshootredact.RedactionList) error {
	redactBytes, err := json.Marshal(redacts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal redactionlist")
	}

	if err := s.saveSupportBundleMetafile(id, "redactions", redactBytes); err != nil {
		return errors.Wrap(err, "faile to save redactions to s3")
	}

	return nil
}

func (s *KOTSStore) saveSupportBundleMetafile(id string, filename string, data []byte) error {
	var gzipped bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipped)
	defer gzipWriter.Close()

	if _, err := gzipWriter.Write(data); err != nil {
		return errors.Wrap(err, "failed to write temp file")
	}
	gzipWriter.Close()

	outputPath := filepath.Join("supportbundles", id, fmt.Sprintf("%s.gz", filename))
	err := filestore.GetStore().WriteArchive(outputPath, bytes.NewReader(gzipped.Bytes()))
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	return nil
}

func (s *KOTSStore) getSupportBundleMetafile(id string, filename string) ([]byte, error) {
	path := filepath.Join("supportbundles", id, fmt.Sprintf("%s.gz", filename))
	bundlePath, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}
	defer os.RemoveAll(bundlePath)

	// Using a file here because Download uses WriterAt type, which bytes.Buffer does not implement.
	gzipFile, err := os.Open(bundlePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open app archive")
	}
	defer gzipFile.Close()

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
		Name           string                  `json:"name"`
		Severity       string                  `json:"severity"`
		Insight        Insight                 `json:"insight"`
		Labels         Labels                  `json:"labels"`
		InvolvedObject *corev1.ObjectReference `json:"involvedObject,omitempty"`
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
			InvolvedObject:  dbInsight.InvolvedObject,
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

func (s *KOTSStore) getSupportBundleFromSecretData(secret *corev1.Secret) (*types.SupportBundle, error) {
	supportBundle := types.SupportBundle{}
	if err := json.Unmarshal(secret.Data["bundle"], &supportBundle); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal support bundle")
	}
	newStatus := getSupportBundleStatus(supportBundle.Status, supportBundle.UpdatedAt)
	if supportBundle.Status == newStatus {
		return &supportBundle, nil
	}

	supportBundle.Status = newStatus
	go s.UpdateSupportBundle(&supportBundle) // must be async to avoid recursive locking

	return &supportBundle, nil
}

func getSupportBundleStatus(lastStatus types.SupportBundleStatus, updatedAt *time.Time) types.SupportBundleStatus {
	if updatedAt == nil || lastStatus != types.BUNDLE_RUNNING {
		return lastStatus
	}

	// If bundle is "running", there should be a loop updating this timestamp.  If not, we call this "failed" after a timeout period
	timeoutTime := time.Now().Add(-10 * time.Second)
	if updatedAt.Before(timeoutTime) {
		return types.BUNDLE_FAILED
	}

	return lastStatus
}
