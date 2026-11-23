package workload

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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

// TestScaleWorkloadRejectsDaemonSetAndUnknownKind pins the scale dispatch: a
// DaemonSet has no replica count and an unknown kind must never fall through
// to a same-name Deployment.
func TestScaleWorkloadRejectsDaemonSetAndUnknownKind(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	for _, kind := range []string{"daemonset", "DaemonSet", "replicaset"} {
		if err := scaleWorkload(context.Background(), clientset, "default", kind, "web", 3, ""); err == nil {
			t.Fatalf("kind %q was scaled instead of rejected", kind)
		}
	}
}

// TestRestartWorkloadPatchesNamedKindOnly regresses the DaemonSet defect: a
// daemonset restart used to patch the same-name Deployment, so the Deployment
// must stay untouched while the DaemonSet's pod template carries the marker.
func TestRestartWorkloadPatchesNamedKindOnly(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}}
	daemonSet := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}}
	clientset := fake.NewSimpleClientset(deployment, daemonSet)
	patch := `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-11-23T00:00:00Z"}}}}}`

	if err := restartWorkload(context.Background(), clientset, "default", "daemonset", "web", "", patch); err != nil {
		t.Fatalf("restart daemonset: %v", err)
	}
	gotDS, err := clientset.AppsV1().DaemonSets("default").Get(context.Background(), "web", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotDS.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] == "" {
		t.Fatal("daemonset was not restarted")
	}
	gotDep, err := clientset.AppsV1().Deployments("default").Get(context.Background(), "web", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotDep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] != "" {
		t.Fatal("restarting the daemonset patched the same-name deployment")
	}
}

func TestAge(t *testing.T) {
	now := time.Now()
	if age(metav1.NewTime(now.Add(-30*time.Second))) == "" {
		t.Fatal("expected non-empty age for 30s")
	}
	if age(metav1.NewTime(now.Add(-2*time.Hour))) == "" {
		t.Fatal("expected non-empty age for 2h")
	}
	if got := age(metav1.Time{}); got != "" {
		t.Fatalf("expected empty age for zero time, got %q", got)
	}
}
