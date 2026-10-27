package backup

import (
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// normalizeUnstructuredJSONValue converts common Go collection values (for
// example []string) to the JSON-native shapes required by Kubernetes
// unstructured objects. The dynamic fake client deep-copies objects using the
// same constraints as a real API server, so keeping this boundary normalized
// avoids runtime panics in both tests and production requests.
func normalizeUnstructuredJSONValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			result[key] = normalizeUnstructuredJSONValue(item)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, item := range typed {
			result[index] = normalizeUnstructuredJSONValue(item)
		}
		return result
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		result := make([]interface{}, rv.Len())
		for index := 0; index < rv.Len(); index++ {
			result[index] = normalizeUnstructuredJSONValue(rv.Index(index).Interface())
		}
		return result
	}
	return value
}

const veleroNamespace = "velero"

var (
	// Velero CRD GroupVersionResource definitions
	backupGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}
	scheduleGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "schedules",
	}
	restoreGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "restores",
	}
	backupStorageLocationGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backupstoragelocations",
	}
)

// VeleroClient wraps access to Velero resources in a K8s cluster.
type VeleroClient struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewVeleroClient creates a Velero client with both standard and dynamic clients.
func NewVeleroClient(clientset kubernetes.Interface, dynamicClient dynamic.Interface) *VeleroClient {
	return &VeleroClient{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

// IsInstalled checks if Velero is installed by checking the velero namespace.
func (c *VeleroClient) IsInstalled(ctx context.Context) (bool, error) {
	if c.clientset == nil {
		return false, fmt.Errorf("clientset is nil")
	}
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, veleroNamespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetVersion retrieves Velero version from the velero deployment.
func (c *VeleroClient) GetVersion(ctx context.Context) (string, error) {
	if c.clientset == nil {
		return "", fmt.Errorf("clientset is nil")
	}
	deployments, err := c.clientset.AppsV1().Deployments(veleroNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, d := range deployments.Items {
		if d.Name == "velero" {
			image := d.Spec.Template.Spec.Containers[0].Image
			return image, nil
		}
	}
	return "", nil
}

// GetRunningPods returns the count of running Velero pods.
func (c *VeleroClient) GetRunningPods(ctx context.Context) (int, error) {
	if c.clientset == nil {
		return 0, fmt.Errorf("clientset is nil")
	}
	pods, err := c.clientset.CoreV1().Pods(veleroNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	running := 0
	for _, p := range pods.Items {
		if p.Status.Phase == "Running" {
			running++
		}
	}
	return running, nil
}

// CreateBackup creates a Velero Backup CRD.
func (c *VeleroClient) CreateBackup(ctx context.Context, name string, spec map[string]interface{}) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	backup := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Backup",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": normalizeUnstructuredJSONValue(spec),
		},
	}

	_, err := c.dynamicClient.Resource(backupGVR).Namespace(veleroNamespace).Create(ctx, backup, metav1.CreateOptions{})
	return err
}

// GetBackup retrieves a Velero Backup CRD status.
func (c *VeleroClient) GetBackup(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(backupGVR).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
}

// ListBackups lists all Velero Backup CRDs.
func (c *VeleroClient) ListBackups(ctx context.Context) (*unstructured.UnstructuredList, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(backupGVR).Namespace(veleroNamespace).List(ctx, metav1.ListOptions{})
}

// DeleteBackup deletes a Velero Backup CRD.
func (c *VeleroClient) DeleteBackup(ctx context.Context, name string) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(backupGVR).Namespace(veleroNamespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// CreateSchedule creates a Velero Schedule CRD.
func (c *VeleroClient) CreateSchedule(ctx context.Context, name string, spec map[string]interface{}) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	schedule := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Schedule",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": normalizeUnstructuredJSONValue(spec),
		},
	}

	_, err := c.dynamicClient.Resource(scheduleGVR).Namespace(veleroNamespace).Create(ctx, schedule, metav1.CreateOptions{})
	return err
}

// GetSchedule retrieves a Velero Schedule CRD.
func (c *VeleroClient) GetSchedule(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(scheduleGVR).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
}

// DeleteSchedule deletes a Velero Schedule CRD.
func (c *VeleroClient) DeleteSchedule(ctx context.Context, name string) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(scheduleGVR).Namespace(veleroNamespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// CreateRestore creates a Velero Restore CRD.
func (c *VeleroClient) CreateRestore(ctx context.Context, name string, spec map[string]interface{}) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	restore := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Restore",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": normalizeUnstructuredJSONValue(spec),
		},
	}

	_, err := c.dynamicClient.Resource(restoreGVR).Namespace(veleroNamespace).Create(ctx, restore, metav1.CreateOptions{})
	return err
}

// GetRestore retrieves a Velero Restore CRD status.
func (c *VeleroClient) GetRestore(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(restoreGVR).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
}

// DeleteRestore deletes a Velero Restore CRD.
func (c *VeleroClient) DeleteRestore(ctx context.Context, name string) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(restoreGVR).Namespace(veleroNamespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// CreateBackupStorageLocation creates a Velero BackupStorageLocation CRD.
func (c *VeleroClient) CreateBackupStorageLocation(ctx context.Context, name string, spec map[string]interface{}) error {
	if c.dynamicClient == nil {
		return fmt.Errorf("dynamic client is nil")
	}

	bsl := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "BackupStorageLocation",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": normalizeUnstructuredJSONValue(spec),
		},
	}

	_, err := c.dynamicClient.Resource(backupStorageLocationGVR).Namespace(veleroNamespace).Create(ctx, bsl, metav1.CreateOptions{})
	return err
}

// GetBackupStorageLocation retrieves a BackupStorageLocation CRD.
func (c *VeleroClient) GetBackupStorageLocation(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is nil")
	}
	return c.dynamicClient.Resource(backupStorageLocationGVR).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
}
