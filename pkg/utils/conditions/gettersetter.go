package conditions

import (
	"fmt"
	"sort"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type GetterSetter interface {
	runtime.Object
	metav1.Object

	// GetConditions returns the list of conditions for a machine API object.
	GetConditions() []v1.Condition

	// SetConditions sets the list of conditions for a machine API object.
	SetConditions([]v1.Condition)
}

func getWrapperObject(from interface{}) GetterSetter {
	switch obj := from.(type) {
	case *v1.Lease:
		return &LeaseWrapper{obj}
	default:
		panic("type is not supported as conditions getter or setter")
	}
}

// lexicographicLess returns true if a condition is less than another with regards to the
// to order of conditions designed for convenience of the consumer, i.e. kubectl.
func lexicographicLess(i, j *v1.Condition) bool {
	return i.Type < j.Type
}

// hasSameState returns true if a condition has the same state of another; state is defined
// by the union of following fields: Type, Status, Reason, Severity and Message (it excludes LastTransitionTime).
func hasSameState(i, j *v1.Condition) bool {
	return i.Type == j.Type &&
		i.Status == j.Status &&
		i.Reason == j.Reason &&
		i.Severity == j.Severity &&
		i.Message == j.Message
}

// Set sets the given condition.
//
// NOTE: If a condition already exists, the LastTransitionTime is updated only if a change is detected
// in any of the following fields: Status, Reason, Severity and Message.
func Set(to interface{}, condition *v1.Condition) {
	if to == nil || condition == nil {
		return
	}

	obj := getWrapperObject(to)

	// Check if the new conditions already exists, and change it only if there is a status
	// transition (otherwise we should preserve the current last transition time)-
	conditions := obj.GetConditions()
	exists := false
	for i := range conditions {
		existingCondition := conditions[i]
		if existingCondition.Type == condition.Type {
			exists = true
			if !hasSameState(&existingCondition, condition) {
				condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
				conditions[i] = *condition
				break
			}
			condition.LastTransitionTime = existingCondition.LastTransitionTime
			break
		}
	}

	// If the condition does not exist, add it, setting the transition time only if not already set
	if !exists {
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
		}
		conditions = append(conditions, *condition)
	}

	// Sorts conditions for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool {
		return lexicographicLess(&conditions[i], &conditions[j])
	})

	obj.SetConditions(conditions)
}

// TrueCondition returns a condition with Status=True and the given type.
func TrueCondition(t v1.ConditionType) *v1.Condition {
	return &v1.Condition{
		Type:   t,
		Status: v1.ConditionTrue,
	}
}

// TrueConditionWithReason returns a condition with Status=True and the given type.
func TrueConditionWithReason(t v1.ConditionType, reason string, messageFormat string, messageArgs ...interface{}) *v1.Condition {
	return &v1.Condition{
		Type:    t,
		Status:  v1.ConditionTrue,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageArgs...),
	}
}

// FalseCondition returns a condition with Status=True and the given type.
func FalseCondition(t v1.ConditionType) *v1.Condition {
	return &v1.Condition{
		Type:   t,
		Status: v1.ConditionFalse,
	}
}

// FalseCondition returns a condition with Status=False and the given type.
func FalseConditionWithReason(t v1.ConditionType, reason string, severity v1.ConditionSeverity, messageFormat string, messageArgs ...interface{}) *v1.Condition {
	return &v1.Condition{
		Type:     t,
		Status:   v1.ConditionFalse,
		Reason:   reason,
		Severity: severity,
		Message:  fmt.Sprintf(messageFormat, messageArgs...),
	}
}

// UnknownCondition returns a condition with Status=Unknown and the given type.
func UnknownCondition(t v1.ConditionType, reason string, messageFormat string, messageArgs ...interface{}) *v1.Condition {
	return &v1.Condition{
		Type:    t,
		Status:  v1.ConditionUnknown,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageArgs...),
	}
}
