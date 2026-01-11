package test

import (
	"fmt"
	"log"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-eng/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-eng/vsphere-capacity-manager/pkg/controller"
)

type shape int64

const (
	SHAPE_SMALL  = shape(1)
	SHAPE_MEDIUM = shape(10)
	SHAPE_LARGE  = shape(100)
)

type lease struct {
	lease v1.Lease
}

// makeProwAnnotations create the annotations for a prow job
func makeProwAnnotations(jobType, jobName string) map[string]string {
	annotations := make(map[string]string)

	switch {
	case jobType == controller.PERIODICAL_JOB_TYPE:
		annotations[controller.PROW_JOB_TYPE_KEY] = controller.PERIODICAL_JOB_TYPE
		annotations[controller.PROW_JOB_KEY] = jobName
		annotations[controller.PROW_BUILD_ID_KEY] = "123456"
		annotations[controller.PROW_GS_BUCKET_KEY] = "test-platform-results"
		annotations[controller.PROW_JOB_URL_PREFIX_KEY] = "https://prow.ci.openshift.org/view/"
	case jobType == controller.PRESUBMIT_JOB_TYPE:
		annotations[controller.PROW_JOB_TYPE_KEY] = controller.PRESUBMIT_JOB_TYPE
		annotations[controller.PROW_JOB_KEY] = jobName
		annotations[controller.PROW_BUILD_ID_KEY] = "654321"
		annotations[controller.PROW_GS_BUCKET_KEY] = "test-platform-results"
		annotations[controller.PROW_JOB_URL_PREFIX_KEY] = "https://prow.ci.openshift.org/view/"
		annotations[controller.GIT_ORG_KEY] = "openshift"
		annotations[controller.GIT_REPO_KEY] = "origin"
		annotations[controller.GIT_PR_KEY] = "987"
	default:
	}

	return annotations
}

// GetLease returns a Lease object for testing
func GetLease() *lease {
	return &lease{
		lease: v1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "sample-lease-",
				Namespace:    "default",
				Labels:       make(map[string]string),
				Annotations:  make(map[string]string),
			},
		},
	}
}

func (r *lease) WithName(name string) *lease {
	r.lease.ObjectMeta.Name = name
	return r
}

func (r *lease) WithShape(shape shape) *lease {
	r.lease.Spec.VCpus = int(16 * int64(shape))
	r.lease.Spec.Memory = int(16 * int64(shape))
	r.lease.Spec.Storage = int(120 * int64(shape))
	r.lease.Spec.Networks = int(1 * int64(shape))

	return r
}

func (r *lease) WithBoskosID(boskosID string) *lease {
	r.lease.Labels[controller.BoskosIdLabel] = boskosID

	return r
}

func (r *lease) WithPool(pool string) *lease {
	r.lease.Spec.RequiredPool = pool
	return r
}

func (r *lease) WithProwAnnotations(jobType, jobName string) *lease {
	// For now, nothing else sets annotations so we'll just set with the prow annotations
	r.lease.Annotations = makeProwAnnotations(jobType, jobName)
	return r
}

func (r *lease) Build() *v1.Lease {
	return &r.lease
}

// IsLeaseOwnedByKinds IsLeaseOwnedByKind checks if the lease is owned by the declared kinds
func IsLeaseOwnedByKinds(lease *v1.Lease, kinds ...string) (bool, error) {
	if lease.Status.Phase != v1.PHASE_FULFILLED {
		return false, fmt.Errorf("lease %s has not been fulfilled", lease.Name)
	}

	for _, kind := range kinds {
		hasKind := false
		for _, ownerRef := range lease.OwnerReferences {
			if ownerRef.Kind == kind {
				hasKind = true
				break
			}
		}
		if !hasKind {
			return false, fmt.Errorf("failed to find %s owner reference for lease %s", kind, lease.Name)
		}
	}

	return true, nil
}

func VerifyCondition(lease *v1.Lease, conditionType v1.ConditionType, status v1.ConditionStatus) bool {
	var targetCondition *v1.Condition

	log.Printf("Checking condition %v for type %v\n", lease.Status.Conditions, conditionType)
	for _, condition := range lease.Status.Conditions {
		log.Printf("Found condition %v", condition.Type)
		if condition.Type == conditionType {
			targetCondition = &condition
			break
		}
	}
	gomega.Expect(targetCondition).ShouldNot(gomega.BeNil(), fmt.Sprintf("condition %s should be set for lease %s", conditionType, lease.Name))
	gomega.Expect(targetCondition.Status).Should(gomega.Equal(status), fmt.Sprintf("condition %s should be %s", conditionType, status))

	return true
}

func VerifyConditionReason(lease *v1.Lease, conditionType v1.ConditionType, status v1.ConditionStatus, reason string) bool {
	var targetCondition *v1.Condition

	log.Printf("Checking condition %v for type %v\n", lease.Status.Conditions, conditionType)
	for _, condition := range lease.Status.Conditions {
		log.Printf("Found condition %v", condition.Type)
		if condition.Type == conditionType {
			targetCondition = &condition
			break
		}
	}
	gomega.Expect(targetCondition).ShouldNot(gomega.BeNil(), fmt.Sprintf("condition %s should be set for lease %s", conditionType, lease.Name))
	gomega.Expect(targetCondition.Status).Should(gomega.Equal(status), fmt.Sprintf("condition %s should be %s", conditionType, status))
	gomega.Expect(targetCondition.Reason).Should(gomega.Equal(reason), fmt.Sprintf("condition %s should have reason '%s'", conditionType, reason))

	return true
}
