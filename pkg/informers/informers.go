package informers

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Start will start the kots informers
// These are not the application level informers, but they are the general purpose KOTS
// informers. For example, we want to watch Velero Backup
// in order to handle out-of-band updates
func Start() error {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	veleroNamespace, err := kotssnapshot.DetectVeleroNamespace(context.TODO(), clientset, util.PodNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}

	veleroClient, err := k8sutil.GetVeleroKubeClient(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	var backupList velerov1.BackupList
	backupWatch, err := veleroClient.Watch(context.TODO(), &backupList, kbclient.InNamespace(veleroNamespace), &kbclient.ListOptions{
		Raw: &metav1.ListOptions{ResourceVersion: "0"},
	})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "failed to watch")
	}

	go func() {
		ch := backupWatch.ResultChan()
		for {
			obj, ok := <-ch // this channel gets closed often
			if !ok {
				if err := Start(); err != nil {
					log.Println("Failed to re-start informers", err)
				}
				break
			}
			if obj.Type == watch.Modified {
				backup, ok := obj.Object.(*velerov1.Backup)
				if !ok {
					logger.Errorf("failed to cast obj to backup")
				}

				if backup.Status.Phase == velerov1.BackupPhaseFailed || backup.Status.Phase == velerov1.BackupPhasePartiallyFailed {
					if backup.Annotations == nil {
						backup.Annotations = map[string]string{}
					}

					_, ok := backup.Annotations["kots.io/support-bundle-requested"]
					if !ok {
						// here.  finally..   request a support bundle for this
						logger.Debugf("requesting support bundle for failed or partially failed backup %s", backup.Name)

						appID, ok := backup.Annotations["kots.io/app-id"]
						if !ok {
							logger.Errorf("failed to find app id anotation on backup")
						}

						backup.Annotations["kots.io/support-bundle-requested"] = time.Now().UTC().Format(time.RFC3339)

						var backup velerov1.Backup
						if err := veleroClient.Update(context.TODO(), &backup); err != nil {
							logger.Error(err)
							continue
						}

						supportBundleID, err := supportbundle.CreateBundleForBackup(appID, backup.Name, backup.Namespace)
						if err != nil {
							logger.Error(err)
							continue
						}

						backup.Annotations["kots.io/support-bundle-id"] = supportBundleID
						if err := veleroClient.Update(context.TODO(), &backup); err != nil {
							logger.Error(err)
							continue
						}
					}
				}
			}
		}
	}()

	return nil
}
