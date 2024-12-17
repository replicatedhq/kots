package plan

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func executeECUpgrade(s store.Store, p *types.Plan, step *types.PlanStep) error {
	a, err := s.GetAppFromSlug(p.AppSlug)
	if err != nil {
		return errors.Wrap(err, "get app from slug")
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return errors.Wrap(err, "parse app license")
	}

	kbClient, err := k8sutil.GetKubeClient(context.Background())
	if err != nil {
		return errors.Wrap(err, "get kubeclient")
	}

	current, err := embeddedcluster.GetCurrentInstallation(context.Background(), kbClient)
	if err != nil {
		return errors.Wrap(err, "get current installation")
	}

	appArchive, err := getAppArchive(p)
	if err != nil {
		return errors.Wrap(err, "get app archive")
	}
	defer os.RemoveAll(appArchive)

	kotsKinds, err := kotsutil.LoadKotsKinds(appArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds from path")
	}

	newInstall := &ecv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ecv1beta1.GroupVersion.String(),
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: current.Spec,
	}
	newInstall.Spec.Artifacts = embeddedcluster.GetArtifactsFromInstallation(kotsKinds.Installation)
	newInstall.Spec.Config = &kotsKinds.EmbeddedClusterConfig.Spec
	newInstall.Spec.LicenseInfo = &ecv1beta1.LicenseInfo{IsDisasterRecoverySupported: license.Spec.IsDisasterRecoverySupported}

	if err := websocket.UpgradeCluster(newInstall); err != nil {
		return errors.Wrap(err, "upgrade cluster")
	}

	return nil
}
