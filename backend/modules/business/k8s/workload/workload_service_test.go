package workload

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNormalizeKind(t *testing.T) {
	cases := map[string]string{
		"deployment":  "deployment",
		"Deployment":  "deployment",
		"statefulset": "statefulset",
		"StatefulSet": "statefulset",
		"daemonset":   "daemonset",
		"DaemonSet":   "daemonset",
		"":            "",
		"unknown":     "",
	}
	for input, want := range cases {
		if got := normalizeKind(input); got != want {
			t.Fatalf("normalizeKind(%q): got %q want %q", input, got, want)
		}
	}
}

func TestWorkloadStatus(t *testing.T) {
	cases := []struct {
		ready, desired int32
		want           string
	}{
		{0, 0, "scaled_down"},
		{3, 3, "ready"},
		{2, 3, "progressing"},
		{5, 3, "ready"},
	}
	for _, c := range cases {
		if got := workloadStatus(c.ready, c.desired); got != c.want {
			t.Fatalf("workloadStatus(%d,%d): got %q want %q", c.ready, c.desired, got, c.want)
		}
	}
}

func TestSafeReplicas(t *testing.T) {
	if got := safeReplicas(nil); got != 0 {
		t.Fatalf("safeReplicas(nil): got %d want 0", got)
	}
	three := int32(3)
	if got := safeReplicas(&three); got != 3 {
		t.Fatalf("safeReplicas(&3): got %d want 3", got)
	}
}

func TestContainerImages(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "nginx:1.25"},
		{Name: "sidecar", Image: "busybox:1.36"},
	}
	images := containerImages(containers)
	if len(images) != 2 || images[0] != "nginx:1.25" || images[1] != "busybox:1.36" {
		t.Fatalf("unexpected images: %v", images)
	}
}

func TestPodRestarts(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{RestartCount: 2},
				{RestartCount: 3},
			},
		},
	}
	if got := podRestarts(pod); got != 5 {
		t.Fatalf("podRestarts: got %d want 5", got)
	}
}

func TestAge(t *testing.T) {
	now := time.Now()
	if got := age(metav1.NewTime(now.Add(-30 * time.Second))); got == "" {
		t.Fatal("expected non-empty age for 30s")
	}
	if got := age(metav1.NewTime(now.Add(-2 * time.Hour))); got == "" {
		t.Fatal("expected non-empty age for 2h")
	}
	if got := age(metav1.Time{}); got != "" {
		t.Fatalf("expected empty age for zero time, got %q", got)
	}
}
