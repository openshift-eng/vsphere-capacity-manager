package test

import (
	"context"
	"fmt"
	"log"
	"strings"

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

		networkReconciler := &controller.NetworkReconciler{
			Client:         mgr.GetClient(),
			UncachedClient: mgr.GetClient(),
			Namespace:      namespaceName,
			OperatorName:   controllerName,
		}
		Expect(networkReconciler.SetupWithManager(mgr)).To(Succeed(), "Reconciler should be able to setup with manager")

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

			// Check to see if initialized
			for _, pool := range knownPools.Items {
				if !pool.Status.Initialized {
					return fmt.Errorf("pool not initialized")
				}
			}

			return nil
		}).Should(Succeed())

		By("Creating test networks")
		for _, network := range networks.Items {
			Expect(k8sClient.Create(ctx, &network)).To(Succeed())
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{
					Namespace: network.Namespace,
					Name:      network.Name,
				}, &network) == nil

			}).Should(BeTrue())
		}

		By("Waiting for networks to enumerate")
		Eventually(func() error {
			knownNetworks := &v1.NetworkList{}
			err = k8sClient.List(ctx, knownNetworks)
			if err != nil {
				return err
			}

			if len(knownNetworks.Items) != len(networks.Items) {
				return fmt.Errorf("network not loaded")
			}

			// Check to see if initialized
			for _, network := range knownNetworks.Items {
				if len(network.Finalizers) == 0 {
					return fmt.Errorf("network not initialized")
				}
			}

			return nil
		}).Should(Succeed())

	}, OncePerOrdered)

	AfterEach(func() {
		By("Stopping the manager")
		mgrCancel()
		// Wait for the mgrDone to be closed, which will happen once the mgr has stopped
		<-mgrDone

		pools := &v1.PoolList{}
		Expect(k8sClient.List(ctx, pools, client.InNamespace(namespaceName))).To(Succeed())
		for _, lease := range pools.Items {
			lease.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, &lease)).To(Succeed())
		}

		leases := &v1.LeaseList{}
		Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
		for _, lease := range leases.Items {
			lease.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, &lease)).To(Succeed())
		}

		networks := &v1.NetworkList{}
		Expect(k8sClient.List(ctx, networks, client.InNamespace(namespaceName))).To(Succeed())
		for _, network := range networks.Items {
			network.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, &network)).To(Succeed())
		}

		Expect(k8sClient.DeleteAllOf(ctx, &v1.Network{}, client.InNamespace(namespaceName))).To(Succeed())
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
			By("lease should be owned by a pool and network", func() {
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
					Expect(leases.Items).To(HaveLen(1))
					result, _ := IsLeaseOwnedByKinds(&leases.Items[0], "Network", "Pool")
					return result
				}).Should(BeTrue())
			})
			By("lease should have a short name", func() {
				Expect(len((&leases.Items[0]).Status.ShortName)).ToNot(Equal(0))
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
			By("lease should be owned by a pool and network", func() {
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
					if len(leases.Items) != 2 {
						return false
					}

					for _, lease := range leases.Items {
						Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
						result, _ := IsLeaseOwnedByKinds(&lease, "Network", "Pool")
						if result == false {
							return false
						}
					}
					return true
				}).Should(BeTrue())
			})
			By("lease should have a short name", func() {
				for _, lease := range leases.Items {
					Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
				}
			})
		})
	})
	It("should fail if no pool is available", func() {
		var leases []*v1.Lease
		By("creating leases", func() {
			for i := 0; i < 4; i++ {
				lease := GetLease().WithShape(SHAPE_MEDIUM).Build()
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
						switch lease.Status.Phase {
						case v1.PHASE_FULFILLED:
							fulfilled++
						case v1.PHASE_PENDING:
							fallthrough
						default:
							pending++
						}
					}
					log.Printf("pending leases: %v", pending)
					log.Printf("fulfilled leases: %v", fulfilled)
					return pending == 3 && fulfilled == 1
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
			By("lease should be owned by a pool and network", func() {
				Eventually(func() bool {
					result, _ := IsLeaseOwnedByKinds(lease, "Pool", "Network")
					return result
				}).Should(BeTrue())
			})
			By("lease should have a short name", func() {
				Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
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
			By("lease should have a short name", func() {
				for _, lease := range leases {
					Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
				}
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
			lease = GetLease().WithShape(SHAPE_SMALL).WithPool("test.com-ibmcloud-vcs-mdcnc-workload-3").Build()
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
			By("lease should be owned by a pool and network", func() {
				Eventually(func() bool {
					result, _ := IsLeaseOwnedByKinds(lease, "Pool", "Network")
					return result
				}).Should(BeTrue())
			})
			By("lease should have a short name", func() {
				Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
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
	It("should acquire two lease in two different vcenters, then delete the leases", func() {
		var lease1 *v1.Lease
		var lease2 *v1.Lease
		By("creating leases", func() {
			// Grab pool from one server
			lease1 = GetLease().WithShape(SHAPE_SMALL).WithPool("test.com-ibmcloud-vcs-ci-workload").WithBoskosID("vsphere-elastic-88").Build()
			Expect(lease1).NotTo(BeNil())
			Expect(k8sClient.Create(ctx, lease1)).To(Succeed())

			// Grab pool from different server
			lease2 = GetLease().WithShape(SHAPE_SMALL).WithPool("test-2.com-ibmcloud-vcs-ci-workload").WithBoskosID("vsphere-elastic-88").Build()
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
			By("lease should be owned by a pool and network", func() {
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, leases, client.InNamespace(namespaceName))).To(Succeed())
					if len(leases.Items) != 2 {
						return false
					}

					for _, lease := range leases.Items {
						result, _ := IsLeaseOwnedByKinds(&lease, "Network", "Pool")
						if result == false {
							return false
						}
					}
					return true
				}).Should(BeTrue())
			})
			By("leases should all have same network", func() {
				lease1Network := lease1.Status.Topology.Networks[0][strings.LastIndex(lease1.Status.Topology.Networks[0], "/"):]
				lease2Network := lease2.Status.Topology.Networks[0][strings.LastIndex(lease2.Status.Topology.Networks[0], "/"):]

				Expect(lease1Network).To(Equal(lease2Network))
			})
			By("lease should have a short name", func() {
				for _, lease := range leases.Items {
					Expect(len(lease.Status.ShortName)).ToNot(Equal(0))
				}
			})
		})

		By("deleting the leases", func() {
			By("by deleting the resource lease", func() {
				Expect(k8sClient.Delete(ctx, lease1)).To(Succeed())
				Expect(k8sClient.Delete(ctx, lease2)).To(Succeed())
			})
			By("waiting for the lease to be deleted", func() {
				Eventually(func() bool {
					return k8sClient.Get(ctx, client.ObjectKeyFromObject(lease1), lease1) != nil
				}).Should(BeTrue())
			})
		})
	})

	// This test is to verify a fix for issue with private ci jobs that were requesting multi zone with multi
	It("should acquire multiple networks for case with multi zone", func() {
		var lease1, lease2 *v1.Lease

		// Create a lease to take all but few of the pool.
		By("creating a resource lease", func() {
			lease1 = GetLease().WithShape(SHAPE_SMALL).WithPool("test.com-ibmcloud-vcs-mdcnc-workload-1").WithBoskosID("vsphere-elastic-88").Build()
			Expect(lease1).NotTo(BeNil())
			lease1.Spec.Networks = 2

			lease2 = GetLease().WithShape(SHAPE_SMALL).WithPool("test.com-ibmcloud-vcs-mdcnc-workload-2").WithBoskosID("vsphere-elastic-88").Build()
			Expect(lease2).NotTo(BeNil())
			lease2.Spec.Networks = 2

			Expect(k8sClient.Create(ctx, lease1)).To(Succeed())
			Expect(k8sClient.Create(ctx, lease2)).To(Succeed())
		})

		// Wait for the start lease to be fulfilled
		By("waiting for leases to be fulfilled", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease1), lease1)
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease2), lease2)

				return lease1.Status.Phase == v1.PHASE_FULFILLED && lease2.Status.Phase == v1.PHASE_FULFILLED
			}).Should(BeTrue())
		})

		// Now delete the leases
		By("deleting the leases", func() {
			By("by deleting the resource lease", func() {
				Expect(k8sClient.Delete(ctx, lease1)).To(Succeed())
				Expect(k8sClient.Delete(ctx, lease2)).To(Succeed())
			})
			By("waiting for the lease to be deleted", func() {
				Eventually(func() bool {
					if k8sClient.Get(ctx, client.ObjectKeyFromObject(lease1), lease1) == nil {
						return false
					}

					return k8sClient.Get(ctx, client.ObjectKeyFromObject(lease2), lease2) != nil
				}).Should(BeTrue())
			})
		})
	})
})
