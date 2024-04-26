package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetAnnotation(meta *metav1.ObjectMeta, key, value string) *metav1.ObjectMeta {
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	meta.Annotations[key] = value
	return meta
}
