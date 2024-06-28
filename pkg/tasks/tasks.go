package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/rqlite/gorqlite"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TaskStatusConfigMapName = `kotsadm-tasks`
	ConfgConfigMapName      = `kotsadm-confg`

	taskCacheTTL = 1 * time.Minute
)

var (
	taskStatusLock   = sync.Mutex{}
	cachedTaskStatus = map[string]*CachedTaskStatus{}
)

type TaskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CachedTaskStatus struct {
	expirationTime time.Time
	taskStatus     TaskStatus
}

func StartTaskMonitor(taskID string, finishedChan <-chan error) {
	go func() {
		var finalError error
		defer func() {
			if finalError == nil {
				if err := ClearTaskStatus(taskID); err != nil {
					logger.Error(errors.Wrapf(err, "failed to clear %s task status", taskID))
				}
			} else {
				errMsg := finalError.Error()
				if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
					errMsg = cause.Error()
				}
				if err := SetTaskStatus(taskID, errMsg, "failed"); err != nil {
					logger.Error(errors.Wrapf(err, "failed to set error on %s task status", taskID))
				}
			}
		}()

		for {
			select {
			case <-time.After(time.Second):
				if err := UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(err)
				}
			case err := <-finishedChan:
				finalError = err
				return
			}
		}
	}()
}

func SetTaskStatus(id string, message string, status string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := cachedTaskStatus[id]
	if cached == nil {
		cached = &CachedTaskStatus{}
		cachedTaskStatus[id] = cached
	}
	cached.taskStatus.Message = message
	cached.taskStatus.Status = status
	cached.taskStatus.UpdatedAt = time.Now()
	cached.expirationTime = time.Now().Add(taskCacheTTL)

	configmap, err := getConfigmap()
	if err != nil {
		if canIgnoreEtcdError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	b, err := json.Marshal(cached.taskStatus)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task status")
	}

	configmap.Data[id] = string(b)

	if err := updateConfigmap(configmap); err != nil {
		if canIgnoreEtcdError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func UpdateTaskStatusTimestamp(id string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := cachedTaskStatus[id]
	if cached != nil {
		cached.taskStatus.UpdatedAt = time.Now()
		cached.expirationTime = time.Now().Add(taskCacheTTL)
	}

	configmap, err := getConfigmap()
	if err != nil {
		if canIgnoreEtcdError(err) && cached != nil {
			return nil
		}
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	data, ok := configmap.Data[id]
	if !ok {
		return nil // copied from s3pgstore
	}

	ts := TaskStatus{}
	if err := json.Unmarshal([]byte(data), &ts); err != nil {
		return errors.Wrap(err, "failed to unmarshal task status")
	}

	ts.UpdatedAt = time.Now()

	b, err := json.Marshal(ts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task status")
	}

	configmap.Data[id] = string(b)

	if err := updateConfigmap(configmap); err != nil {
		if canIgnoreEtcdError(err) && cached != nil {
			return nil
		}
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func ClearTaskStatus(id string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	defer delete(cachedTaskStatus, id)

	configmap, err := getConfigmap()
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	_, ok := configmap.Data[id]
	if !ok {
		return nil // copied from s3pgstore
	}

	delete(configmap.Data, id)

	if err := updateConfigmap(configmap); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func GetTaskStatus(id string) (string, string, error) {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := cachedTaskStatus[id]
	if cached != nil && time.Now().Before(cached.expirationTime) {
		return cached.taskStatus.Status, cached.taskStatus.Message, nil
	}

	if cached == nil {
		cached = &CachedTaskStatus{
			expirationTime: time.Now().Add(taskCacheTTL),
		}
		cachedTaskStatus[id] = cached
	}

	configmap, err := getConfigmap()
	if err != nil {
		if canIgnoreEtcdError(err) && cached != nil {
			return cached.taskStatus.Status, cached.taskStatus.Message, nil
		}
		return "", "", errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		return "", "", nil
	}

	marshalled, ok := configmap.Data[id]
	if !ok {
		return "", "", nil
	}

	ts := TaskStatus{}
	if err := json.Unmarshal([]byte(marshalled), &ts); err != nil {
		return "", "", errors.Wrap(err, "error unmarshalling task status")
	}

	if ts.UpdatedAt.Before(time.Now().Add(-10 * time.Second)) {
		return "", "", nil
	}

	cached.taskStatus = ts

	return ts.Status, ts.Message, nil
}

func getConfigmap() (*corev1.ConfigMap, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingConfigmap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), TaskStatusConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configmap := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      TaskStatusConfigMapName,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{},
		}

		createdConfigmap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), &configmap, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create configmap")
		}

		return createdConfigmap, nil
	}

	return existingConfigmap, nil
}

func updateConfigmap(configmap *corev1.ConfigMap) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func canIgnoreEtcdError(err error) bool {
	if err == nil {
		return true
	}

	if strings.Contains(err.Error(), "connection refused") {
		return true
	}

	if strings.Contains(err.Error(), "request timed out") {
		return true
	}

	if strings.Contains(err.Error(), "EOF") {
		return true
	}

	return false
}

func MigrateTasksFromRqlite() error {
	db := persistence.MustGetDBSession()

	query := `select updated_at, current_message, status from api_task_status`
	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to select tasks for migration: %v: %v", err, rows.Err)
	}

	taskCm, err := getConfigmap()
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if taskCm.Data == nil {
		taskCm.Data = map[string]string{}
	}

	for rows.Next() {
		var id string
		var status gorqlite.NullString
		var message gorqlite.NullString

		ts := TaskStatus{}
		if err := rows.Scan(&id, &ts.UpdatedAt, &message, &status); err != nil {
			return errors.Wrap(err, "failed to scan task status")
		}

		if status.Valid {
			ts.Status = status.String
		}
		if message.Valid {
			ts.Message = message.String
		}

		b, err := json.Marshal(ts)
		if err != nil {
			return errors.Wrap(err, "failed to marshal task status")
		}

		taskCm.Data[id] = string(b)
	}

	if err := updateConfigmap(taskCm); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	query = `delete from api_task_status`
	if wr, err := db.WriteOne(query); err != nil {
		return fmt.Errorf("failed to delete tasks from db: %v: %v", err, wr.Err)
	}

	return nil
}
