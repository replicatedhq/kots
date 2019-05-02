package shipwatch

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	"golang.org/x/net/context"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
var watchJobKey = types.NamespacedName{Name: "foo-watch", Namespace: "default"}

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &shipv1beta1.ShipWatch{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
	defer close(StartTestManager(mgr, g))

	// Create the ShipWatch object and expect the Reconcile and Deployment to be created
	err = c.Create(context.TODO(), instance)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	watchJob := &batchv1.Job{}
	g.Eventually(func() error { return c.Get(context.TODO(), watchJobKey, watchJob) }, timeout).
		Should(gomega.Succeed())

	// Delete the watch Job and expect Reconcile to be called for Deployment deletion
	err = c.Delete(context.TODO(), watchJob)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), watchJobKey, watchJob) }, timeout).
		Should(gomega.Succeed())

	// Manually delete Deployment since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), watchJob)).To(gomega.Succeed())

}
