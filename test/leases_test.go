package test

import (
	"context"
	"fmt"
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/controller"
	"k8s.io/apimachinery/pkg/types"
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

		leaseReconciler := &controller.LeaseReconciler{
			Client:         mgr.GetClient(),
			UncachedClient: mgr.GetClient(),
			Namespace:      namespaceName,
			OperatorName:   controllerName,
		}
		Expect(leaseReconciler.SetupWithManager(mgr)).To(Succeed(), "Reconciler should be able to setup with manager")

		poolReconciler := &controller.PoolReconciler{
			Client:         mgr.GetClient(),
			UncachedClient: mgr.GetClient(),
			Namespace:      namespaceName,
			OperatorName:   controllerName,
		}
		Expect(poolReconciler.SetupWithManager(mgr)).To(Succeed(), "Reconciler should be able to setup with manager")

		By("Starting the manager")
		var mgrCtx context.Context
		mgrCtx, mgrCancel = context.WithCancel(context.Background())
		mgrDone = make(chan struct{})

		go func() {
			defer GinkgoRecover()
			defer close(mgrDone)

			Expect(mgr.Start(mgrCtx)).To(Succeed())
		}()

		By("Creating test pools")
		for idx, pool := range pools.Items {
			Expect(k8sClient.Create(ctx, &pool)).To(Succeed())
			poolStatus := pools.Items[idx].Status.DeepCopy()
			Eventually(func() bool {
				err = k8sClient.Get(ctx, types.NamespacedName{
					Namespace: pool.Namespace,
					Name:      pool.Name,
				}, &pool)
				poolStatus.DeepCopyInto(&pool.Status)
				err = k8sClient.Status().Update(ctx, &pool)
				return err == nil
			}).Should(BeTrue())
		}

		By("Waiting for pools to enumerate")
		Eventually(func() error {
			knownPools := &v1.PoolList{}
			err = k8sClient.List(ctx, knownPools)
			if err != nil {
				return err
			}

			if len(knownPools.Items) != len(pools.Items) {
				return fmt.Errorf("pools not loaded")
			}
			return nil
		}).Should(Succeed())
	}, OncePerOrdered)

	AfterEach(func() {
		By("Stopping the manager")
		mgrCancel()
		// Wait for the mgrDone to be closed, which will happen once the mgr has stopped
		<-mgrDone

		leases := &v1.LeaseList{}
		Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
		for _, lease := range leases.Items {
			lease.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, &lease)).To(Succeed())
		}

		Expect(k8sClient.DeleteAllOf(ctx, &v1.Lease{}, client.InNamespace(namespaceName))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &v1.Pool{}, client.InNamespace(namespaceName))).To(Succeed())

	}, OncePerOrdered)
	It("should acquire single lease", func() {
		var req *v1.Lease
		By("creating a resource lease", func() {
			req = GetLease().WithShape(SHAPE_SMALL).Build()
			Expect(req).NotTo(BeNil())

			Expect(k8sClient.Create(ctx, req)).To(Succeed())
		})

		By("waiting for lease to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(req), req)
				return req.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		leases := &v1.LeaseList{}
		By("checking the lease", func() {
			By("associated pool should reflect the resources claimed by the lease", func() {
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
					Expect(leases.Items).To(HaveLen(1))

					result, _ := DoesLeaseHavePool(&leases.Items[0])
					return result
				}).Should(BeTrue())
			})
		})
	})

	It("should acquire 2 leases", func() {
		var lease1 *v1.Lease
		var lease2 *v1.Lease

		By("creating a resource lease", func() {
			lease1 = GetLease().WithShape(SHAPE_SMALL).Build()
			Expect(lease1).NotTo(BeNil())
			Expect(k8sClient.Create(ctx, lease1)).To(Succeed())

			lease2 = GetLease().WithShape(SHAPE_SMALL).Build()
			Expect(lease2).NotTo(BeNil())
			Expect(k8sClient.Create(ctx, lease2)).To(Succeed())
		})

		By("waiting for leases to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease1), lease1)
				if lease1.Status.Phase != v1.PHASE_FULFILLED {
					return false
				}

				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease2), lease2)
				return lease2.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		leases := &v1.LeaseList{}
		By("checking the lease", func() {
			By("associated pool should reflect the resources claimed by the lease", func() {
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
					if len(leases.Items) != 2 {
						return false
					}

					for _, lease := range leases.Items {
						result, _ := DoesLeaseHavePool(&lease)
						if result == false {
							return false
						}
					}
					return true
				}).Should(BeTrue())
			})
		})
	})
	It("should fail if no pool is available", func() {
		var leases []*v1.Lease
		By("creating leases", func() {
			for i := 0; i < 3; i++ {
				lease := GetLease().WithShape(SHAPE_SMALL).Build()
				Expect(lease).NotTo(BeNil())
				Expect(k8sClient.Create(ctx, lease)).To(Succeed())
				leases = append(leases, lease)
			}
		})

		By("checking the lease", func() {
			By("checking that one of the three leases never gets fulfilled", func() {
				Eventually(func() bool {
					// Check that at least one lease is not fulfilled
					pending := 0
					fulfilled := 0
					for _, lease := range leases {
						err := k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease)
						if err != nil {
							log.Printf("unable to get lease: %v", err)
							return false
						}
						if len(lease.Status.Phase) == 0 {
							return false
						}
						switch lease.Status.Phase {
						case v1.PHASE_FULFILLED:
							fulfilled++
						case v1.PHASE_PENDING:
							pending++
						}
					}
					log.Printf("pending leases: %v", pending)
					log.Printf("fulfilled leases: %v", fulfilled)
					return pending == 1 && fulfilled == 2
				}).Should(BeTrue())
			})
		})
	})
	It("should acquire single lease, then delete it", func() {
		var lease *v1.Lease
		By("creating a resource lease", func() {
			lease = GetLease().WithShape(SHAPE_SMALL).Build()
			Expect(lease).NotTo(BeNil())

			Expect(k8sClient.Create(ctx, lease)).To(Succeed())
		})

		By("waiting for lease to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease)
				return lease.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		By("checking the lease", func() {
			By("associated pool should reflect the resources claimed by the lease", func() {
				Eventually(func() bool {
					result, _ := DoesLeaseHavePool(lease)
					return result
				}).Should(BeTrue())
			})
		})

		By("deleting the lease", func() {
			By("by deleting the resource lease", func() {
				Expect(k8sClient.Delete(ctx, lease)).To(Succeed())
			})
			By("waiting for the lease to be deleted", func() {
				Eventually(func() bool {
					return k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease) != nil
				}).Should(BeTrue())
			})
		})
	})
	It("should acquire two leases, then delete them", func() {
		var leases []*v1.Lease
		By("creating leases", func() {
			for i := 0; i < 2; i++ {
				lease := GetLease().WithShape(SHAPE_SMALL).Build()
				Expect(lease).NotTo(BeNil())
				Expect(k8sClient.Create(ctx, lease)).To(Succeed())
				leases = append(leases, lease)
			}
		})

		By("waiting for leases to be fulfilled", func() {
			Eventually(func() bool {
				for _, lease := range leases {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease)
					if err != nil {
						return false
					}
					if lease.Status.Phase != v1.PHASE_FULFILLED {
						return false
					}
				}
				return true
			}).Should(BeTrue())
		})

		By("checking the leases", func() {
			By("associated pool should reflect the resources claimed by the leases", func() {
				Eventually(func() bool {
					fulfilled := 0
					for _, lease := range leases {
						err := k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease)
						if err != nil {
							log.Printf("unable to get lease: %v", err)
							return false
						}
						if len(lease.Status.Phase) == 0 {
							return false
						}
						switch lease.Status.Phase {
						case v1.PHASE_FULFILLED:
							fulfilled++
						}
					}
					return fulfilled == 2

				}).Should(BeTrue())
			})
		})

		By("deleting the leases", func() {
			By("by deleting the resource lease", func() {
				for _, lease := range leases {
					Expect(k8sClient.Delete(ctx, lease)).To(Succeed())
				}
			})
			By("waiting for the leases to be deleted", func() {
				Eventually(func() bool {
					for _, lease := range leases {
						if k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease) == nil {
							return false
						}
					}
					return true
				}).Should(BeTrue())
			})
		})
	})
	It("should acquire a lease in a non-default pool, then delete the lease", func() {
		var lease *v1.Lease
		By("creating a resource lease", func() {
			lease = GetLease().WithShape(SHAPE_SMALL).WithPool("sample-zonal-pool-0").Build()
			Expect(lease).NotTo(BeNil())
			Expect(k8sClient.Create(ctx, lease)).To(Succeed())
		})

		By("waiting for the leases to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease)
				return lease.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		By("checking the leases", func() {
			By("associated pool should reflect the resources claimed by the leases", func() {
				Eventually(func() bool {
					result, _ := DoesLeaseHavePool(lease)
					return result
				}).Should(BeTrue())
			})
		})

		By("deleting the leases", func() {
			By("by deleting the resource lease", func() {
				Expect(k8sClient.Delete(ctx, lease)).To(Succeed())
			})
			By("waiting for the lease to be deleted", func() {
				Eventually(func() bool {
					return k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), lease) != nil
				}).Should(BeTrue())
			})
		})
	})
})
