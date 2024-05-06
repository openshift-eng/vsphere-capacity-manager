package test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var (
	// specify testEnv configuration
	testEnv    *envtest.Environment
	cfg        *rest.Config
	k8sClient  client.Client
	testScheme *runtime.Scheme
	ctx        = context.Background()
	pools      = v1.PoolList{
		Items: []v1.Pool{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vspherecapacitymanager.splat.io/v1",
					Kind:       "Pool",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-pool-0",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      20,
					Memory:     200,
					Storage:    1000,
					Server:     "vcs8e-vc.ocp2.dev.cluster.com",
					Datacenter: "dc-0",
					Cluster:    "cluster-0",
					Datastore:  "datastore-0",
					Exclude:    false,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-pool-1",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      20,
					Memory:     200,
					Storage:    2000,
					Server:     "vcenter.ibmc.devcluster.openshift.com",
					Datacenter: "dc-1",
					Cluster:    "cluster-1",
					Datastore:  "datastore-1",
					Exclude:    false,
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vspherecapacitymanager.splat.io/v1",
					Kind:       "Pool",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-zonal-pool-0",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      40,
					Memory:     400,
					Storage:    4000,
					Server:     "vcenter.devqe.ibmc.devcluster.openshift.com",
					Datacenter: "dc-2",
					Cluster:    "cluster-2",
					Datastore:  "datastore-2",
					Exclude:    true,
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vspherecapacitymanager.splat.io/v1",
					Kind:       "Pool",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-zonal-pool-1",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      20,
					Memory:     200,
					Storage:    2000,
					Server:     "v8c-2-vcenter.ocp2.dev.cluster.com",
					Datacenter: "dc-3",
					Cluster:    "cluster-3",
					Datastore:  "datastore-3",
					Exclude:    true,
				},
			},
		},
	}
)

func TestLeases(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Leases Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	var err error
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	testScheme = scheme.Scheme
	Expect(v1.Install(testScheme)).To(Succeed())

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	SetDefaultEventuallyTimeout(10 * time.Second)

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	komega.SetClient(k8sClient)
	komega.SetContext(ctx)
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})
