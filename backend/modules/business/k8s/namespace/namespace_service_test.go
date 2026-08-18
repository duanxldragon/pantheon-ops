package namespace

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToNamespaceItem(t *testing.T) {
	created := time.Now()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default",
			Labels:            map[string]string{"env": "prod"},
			CreationTimestamp: metav1.NewTime(created),
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}

	item := toNamespaceItem(ns)
	if item.Name != "default" || item.Status != "Active" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Labels["env"] != "prod" {
		t.Fatalf("expected labels, got %v", item.Labels)
	}
	if item.CreationTimestamp == "" {
		t.Fatal("expected creation timestamp to be formatted")
	}
}
