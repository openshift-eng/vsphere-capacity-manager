package test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

var _ = Describe("Lease management", func() {
	It("should acquire lease", func() {
		var req *v1.ResourceRequest
		By("creating a resource request", func() {
			req = GetResourceRequest().WithShape(SHAPE_SMALL).Build()
			Expect(req).NotTo(BeNil())

			Expect(k8sClient.Create(ctx, req)).To(Succeed())
		})

		By("waiting for lease to be fulfilled", func() {

		})
	})
})
