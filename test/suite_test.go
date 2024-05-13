package test

import (
	"context"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
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
		Items: []v1.Pool{},
	}
	networks = v1.NetworkList{
		Items: []v1.Network{},
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

	dirEntries, err := os.ReadDir("./manifests")
	Expect(err).NotTo(HaveOccurred())
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join("./manifests", entry.Name()))
		Expect(err).NotTo(HaveOccurred())

		if strings.HasPrefix(entry.Name(), "network-ci-vlan") {
			network := v1.Network{}
			err = yaml.Unmarshal(content, &network)
			Expect(err).NotTo(HaveOccurred())
			network.Namespace = "default"
			network.Name = strings.ToLower(network.Name)
			networks.Items = append(networks.Items, network)
		} else if strings.HasPrefix(entry.Name(), "pool-") {
			pool := v1.Pool{}
			err = yaml.Unmarshal(content, &pool)
			Expect(err).NotTo(HaveOccurred())
			pool.Namespace = "default"
			pool.Name = strings.ToLower(pool.Name)
			pools.Items = append(pools.Items, pool)
		}
	}

	komega.SetClient(k8sClient)
	komega.SetContext(ctx)
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})
