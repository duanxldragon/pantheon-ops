package release

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// previewWorkloadContext projects the exact observed workload reviewed by the
// image guard. These fields stay unchanged by image updates; configuration
// values, names of secret keys/references, commands and annotations stay private.
func previewWorkloadContext(item metav1.Object, containerName string) []PreviewChange {
	var pod corev1.PodSpec
	var desired *int32
	switch target := item.(type) {
	case *appsv1.Deployment:
		pod, desired = target.Spec.Template.Spec, target.Spec.Replicas
	case *appsv1.StatefulSet:
		pod, desired = target.Spec.Template.Spec, target.Spec.Replicas
	case *appsv1.DaemonSet:
		pod = target.Spec.Template.Spec
	default:
		return nil
	}
	changes := make([]PreviewChange, 0, 9)
	unchanged := func(field, value string) {
		changes = append(changes, PreviewChange{ID: field, Field: field, Kind: "unchanged", Before: value, After: value})
	}
	if desired != nil {
		unchanged("replicas", strconv.FormatInt(int64(*desired), 10))
	}
	unchanged("containers", strconv.Itoa(len(pod.Containers)))
	unchanged("volumes", strconv.Itoa(len(pod.Volumes)))
	for _, container := range pod.Containers {
		if containerName != "" && container.Name != containerName {
			continue
		}
		unchanged("environmentEntries", strconv.Itoa(len(container.Env)))
		unchanged("configurationSources", strconv.Itoa(len(container.EnvFrom)))
		for _, value := range []struct {
			field string
			list  corev1.ResourceList
			name  corev1.ResourceName
		}{
			{"cpuRequest", container.Resources.Requests, corev1.ResourceCPU},
			{"memoryRequest", container.Resources.Requests, corev1.ResourceMemory},
			{"cpuLimit", container.Resources.Limits, corev1.ResourceCPU},
			{"memoryLimit", container.Resources.Limits, corev1.ResourceMemory},
		} {
			if quantity, ok := value.list[value.name]; ok {
				unchanged(value.field, quantity.String())
			}
		}
		break // The image contract selects the first container when name is absent.
	}
	return changes
}
