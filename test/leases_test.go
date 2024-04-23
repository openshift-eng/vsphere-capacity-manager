package test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/controller"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	"k8s.io/klog/v2/textlogger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var _ = Describe("Lease management", func() {
	var mgrCancel context.CancelFunc
	var mgrDone chan struct{}

	namespaceName := "default"

	const controllerName = "lease-reconiler"

	BeforeEach(func() {
		By("starting the lease reconciler")

		logger := textlogger.NewLogger(textlogger.NewConfig())
		ctrl.SetLogger(logger)

		var err error
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: testScheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
			WebhookServer: webhook.NewServer(webhook.Options{
				Port:    testEnv.WebhookInstallOptions.LocalServingPort,
				Host:    testEnv.WebhookInstallOptions.LocalServingHost,
				CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
			}),
		})
		Expect(err).ToNot(HaveOccurred(), "Manager should be able to be created")

		poolReconciler := &controller.PoolReconciler{
			Client:         mgr.GetClient(),
			UncachedClient: mgr.GetClient(),
			Namespace:      namespaceName,
			OperatorName:   controllerName,
		}
		Expect(poolReconciler.SetupWithManager(mgr)).To(Succeed(), "Reconciler should be able to setup with manager")

		leaseReconciler := &controller.LeaseReconciler{
			Client:         mgr.GetClient(),
			UncachedClient: mgr.GetClient(),
			Namespace:      namespaceName,
			OperatorName:   controllerName,
		}
		Expect(leaseReconciler.SetupWithManager(mgr)).To(Succeed(), "Reconciler should be able to setup with manager")

		By("Starting the manager")
		var mgrCtx context.Context
		mgrCtx, mgrCancel = context.WithCancel(context.Background())
		mgrDone = make(chan struct{})

		go func() {
			defer GinkgoRecover()
			defer close(mgrDone)

			Expect(mgr.Start(mgrCtx)).To(Succeed())
		}()

		By("Waiting for pools to enumerate")
		Eventually(func() error {
			if resources.GetPools() == nil || resources.GetPoolCount() != len(pools.Items) {
				return fmt.Errorf("pools not loaded")
			}
			return nil
		}).Should(Succeed())

		//resources.ReconcileSubnets(pools.)

	}, OncePerOrdered)

	AfterEach(func() {
		By("Stopping the manager")
		mgrCancel()
		// Wait for the mgrDone to be closed, which will happen once the mgr has stopped
		<-mgrDone

		Expect(k8sClient.DeleteAllOf(ctx, &v1.ResourceRequest{}, client.InNamespace(namespaceName))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &v1.Lease{}, client.InNamespace(namespaceName))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &v1.Pool{}, client.InNamespace(namespaceName))).To(Succeed())

	}, OncePerOrdered)
	It("should acquire lease", func() {
		var req *v1.ResourceRequest
		By("creating a resource request", func() {
			req = GetResourceRequest().WithShape(SHAPE_SMALL).Build()
			Expect(req).NotTo(BeNil())

			Expect(k8sClient.Create(ctx, req)).To(Succeed())
		})

		By("waiting for lease to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(req), req)
				return req.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		By("checking the lease", func() {
			leases := &v1.LeaseList{}
			Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
			Expect(leases.Items).To(HaveLen(1))
		})
	})
})
