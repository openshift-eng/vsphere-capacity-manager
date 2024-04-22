package test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	// specify testEnv configuration
	testEnv *envtest.Environment
)

func TestLeases(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Leases Suite")
}

var _ = BeforeSuite(func(done Done) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}
	close(done)
}, 60)

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})
