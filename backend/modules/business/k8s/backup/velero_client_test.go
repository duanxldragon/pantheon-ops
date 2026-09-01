package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVeleroClient_IsInstalled(t *testing.T) {
	tests := []struct {
		name      string
		namespace *corev1.Namespace
		want      bool
		wantErr   bool
	}{
		{
			name: "velero installed",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "velero",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "velero not installed",
			namespace: nil,
			want:      false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			if tt.namespace != nil {
				_, err := clientset.CoreV1().Namespaces().Create(context.Background(), tt.namespace, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
			vc := NewVeleroClient(clientset, dynamicClient)

			got, err := vc.IsInstalled(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVeleroClient_CreateBackup(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset()

	vc := NewVeleroClient(clientset, dynamicClient)

	spec := map[string]interface{}{
		"includedNamespaces": []string{"default", "kube-system"},
		"ttl":                "720h",
	}

	err := vc.CreateBackup(context.Background(), "test-backup", spec)
	assert.NoError(t, err)

	// Verify backup was created
	backup, err := vc.GetBackup(context.Background(), "test-backup")
	require.NoError(t, err)
	assert.Equal(t, "test-backup", backup.GetName())
}

func TestVeleroClient_CreateSchedule(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset()

	vc := NewVeleroClient(clientset, dynamicClient)

	spec := map[string]interface{}{
		"schedule": "0 1 * * *",
		"template": map[string]interface{}{
			"includedNamespaces": []string{"default"},
		},
	}

	err := vc.CreateSchedule(context.Background(), "daily-backup", spec)
	assert.NoError(t, err)

	// Verify schedule was created
	schedule, err := vc.GetSchedule(context.Background(), "daily-backup")
	require.NoError(t, err)
	assert.Equal(t, "daily-backup", schedule.GetName())
}

func TestVeleroClient_CreateRestore(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	clientset := fake.NewSimpleClientset()

	vc := NewVeleroClient(clientset, dynamicClient)

	spec := map[string]interface{}{
		"backupName": "test-backup",
		"includedNamespaces": []string{"default"},
	}

	err := vc.CreateRestore(context.Background(), "test-restore", spec)
	assert.NoError(t, err)

	// Verify restore was created
	restore, err := vc.GetRestore(context.Background(), "test-restore")
	require.NoError(t, err)
	assert.Equal(t, "test-restore", restore.GetName())
}

func TestVeleroClient_DeleteBackup(t *testing.T) {
	scheme := runtime.NewScheme()

	// Create a backup first
	backup := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Backup",
			"metadata": map[string]interface{}{
				"name":      "test-backup",
				"namespace": "velero",
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, backup)
	clientset := fake.NewSimpleClientset()

	vc := NewVeleroClient(clientset, dynamicClient)

	// Delete backup
	err := vc.DeleteBackup(context.Background(), "test-backup")
	assert.NoError(t, err)

	// Verify backup was deleted
	_, err = vc.GetBackup(context.Background(), "test-backup")
	assert.Error(t, err)
}

func TestVeleroClient_GetRunningPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "velero",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "velero-1",
				Namespace: "velero",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "velero-2",
				Namespace: "velero",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
	)

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	vc := NewVeleroClient(clientset, dynamicClient)

	count, err := vc.GetRunningPods(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Only one pod is Running
}
