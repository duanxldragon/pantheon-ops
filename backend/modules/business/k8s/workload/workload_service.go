package workload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/namespace"
	"pantheon-base/pkg/common"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	syncTimeout             = 30 * time.Second
	workloadKindDeployment  = "deployment"
	workloadKindStatefulSet = "statefulset"
	workloadKindDaemonSet   = "daemonset"
)

// WorkloadService reads and mutates Kubernetes workloads through client-go.
//
//nolint:revive // retained as the public service name for this package.
type WorkloadService struct {
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewWorkloadService creates a Kubernetes workload service.
func NewWorkloadService(clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *WorkloadService {
	return &WorkloadService{clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// List returns deployments, statefulsets and daemonsets for a cluster,
// optionally filtered by namespace and kind. Kind accepts "deployment",
// "statefulset" or "daemonset" (case-insensitive); an empty kind returns all.
func (s *WorkloadService) List(query WorkloadListQuery, dataScope *common.DataScopeReq) (*WorkloadListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(query.ClusterID, dataScope)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	ns := query.Namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	items := make([]WorkloadItem, 0)
	options := metav1.ListOptions{Limit: normalizeListLimit(query.Limit), Continue: query.ContinueToken}
	continueToken := ""
	switch normalizeKind(query.Kind) {
	case workloadKindDeployment:
		list, err := clientset.AppsV1().Deployments(ns).List(ctx, options)
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		continueToken = list.Continue
		for i := range list.Items {
			d := &list.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "Deployment",
				Name:            d.Name,
				Namespace:       d.Namespace,
				Replicas:        safeReplicas(d.Spec.Replicas),
				ReadyReplicas:   d.Status.ReadyReplicas,
				Images:          containerImages(d.Spec.Template.Spec.Containers),
				Status:          workloadStatus(d.Status.ReadyReplicas, safeReplicas(d.Spec.Replicas)),
				Age:             age(d.CreationTimestamp),
				ResourceVersion: d.ResourceVersion,
			})
		}
	case workloadKindStatefulSet:
		list, err := clientset.AppsV1().StatefulSets(ns).List(ctx, options)
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		continueToken = list.Continue
		for i := range list.Items {
			st := &list.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "StatefulSet",
				Name:            st.Name,
				Namespace:       st.Namespace,
				Replicas:        safeReplicas(st.Spec.Replicas),
				ReadyReplicas:   st.Status.ReadyReplicas,
				Images:          containerImages(st.Spec.Template.Spec.Containers),
				Status:          workloadStatus(st.Status.ReadyReplicas, safeReplicas(st.Spec.Replicas)),
				Age:             age(st.CreationTimestamp),
				ResourceVersion: st.ResourceVersion,
			})
		}
	case workloadKindDaemonSet:
		list, err := clientset.AppsV1().DaemonSets(ns).List(ctx, options)
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		continueToken = list.Continue
		for i := range list.Items {
			ds := &list.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "DaemonSet",
				Name:            ds.Name,
				Namespace:       ds.Namespace,
				Replicas:        ds.Status.DesiredNumberScheduled,
				ReadyReplicas:   ds.Status.NumberReady,
				Images:          containerImages(ds.Spec.Template.Spec.Containers),
				Status:          workloadStatus(ds.Status.NumberReady, ds.Status.DesiredNumberScheduled),
				Age:             age(ds.CreationTimestamp),
				ResourceVersion: ds.ResourceVersion,
			})
		}
	default:
		deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, options)
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		for i := range deployments.Items {
			d := &deployments.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "Deployment",
				Name:            d.Name,
				Namespace:       d.Namespace,
				Replicas:        safeReplicas(d.Spec.Replicas),
				ReadyReplicas:   d.Status.ReadyReplicas,
				Images:          containerImages(d.Spec.Template.Spec.Containers),
				Status:          workloadStatus(d.Status.ReadyReplicas, safeReplicas(d.Spec.Replicas)),
				Age:             age(d.CreationTimestamp),
				ResourceVersion: d.ResourceVersion,
			})
		}
		statefulsets, err := clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		for i := range statefulsets.Items {
			st := &statefulsets.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "StatefulSet",
				Name:            st.Name,
				Namespace:       st.Namespace,
				Replicas:        safeReplicas(st.Spec.Replicas),
				ReadyReplicas:   st.Status.ReadyReplicas,
				Images:          containerImages(st.Spec.Template.Spec.Containers),
				Status:          workloadStatus(st.Status.ReadyReplicas, safeReplicas(st.Spec.Replicas)),
				Age:             age(st.CreationTimestamp),
				ResourceVersion: st.ResourceVersion,
			})
		}
		daemonsets, err := clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.New("k8s.workload.list_failed")
		}
		for i := range daemonsets.Items {
			ds := &daemonsets.Items[i]
			items = append(items, WorkloadItem{
				Kind:            "DaemonSet",
				Name:            ds.Name,
				Namespace:       ds.Namespace,
				Replicas:        ds.Status.DesiredNumberScheduled,
				ReadyReplicas:   ds.Status.NumberReady,
				Images:          containerImages(ds.Spec.Template.Spec.Containers),
				Status:          workloadStatus(ds.Status.NumberReady, ds.Status.DesiredNumberScheduled),
				Age:             age(ds.CreationTimestamp),
				ResourceVersion: ds.ResourceVersion,
			})
		}
	}

	if strings.TrimSpace(query.Kind) == "" {
		// Multi-kind lists cannot safely compose Kubernetes continuation tokens;
		// callers must select a kind to continue beyond this bounded first page.
		continueToken = ""
	}
	return &WorkloadListResponse{Items: items, Total: len(items), ContinueToken: continueToken}, nil
}

func normalizeListLimit(limit int64) int64 {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

// GetPods returns the pods owned by a workload, matched by a label selector.
func (s *WorkloadService) GetPods(clusterID uint64, namespace, kind, name string, dataScope *common.DataScopeReq) (*PodListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	selector, err := s.workloadSelector(ctx, clientset, namespace, kind, name)
	if err != nil {
		return nil, err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, errors.New("k8s.workload.pods_failed")
	}

	items := make([]PodItem, 0, len(podList.Items))
	for i := range podList.Items {
		p := &podList.Items[i]
		items = append(items, PodItem{
			Name:      p.Name,
			Status:    string(p.Status.Phase),
			NodeName:  p.Spec.NodeName,
			Restarts:  podRestarts(p),
			CreatedAt: p.CreationTimestamp.Format(time.RFC3339),
		})
	}
	return &PodListResponse{Items: items, Total: len(items)}, nil
}

// Scale adjusts the replica count of a deployment or statefulset.
func (s *WorkloadService) Scale(clusterID uint64, namespace, kind, name string, replicas int32, dataScope *common.DataScopeReq) error {
	return s.ScaleWithResourceVersion(clusterID, namespace, kind, name, replicas, "", dataScope)
}

func (s *WorkloadService) ScaleWithResourceVersion(clusterID uint64, namespace, kind, name string, replicas int32, expectedResourceVersion string, dataScope *common.DataScopeReq) error {
	if s.namespaceSvc == nil {
		return errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "workload:scale"); err != nil {
		return err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	switch normalizeKind(kind) {
	case workloadKindStatefulSet:
		st, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.New("k8s.workload.not_found")
		}
		if expectedResourceVersion != "" && st.ResourceVersion != expectedResourceVersion {
			return common.NewConflict("k8s.workload.resource_version_conflict")
		}
		st.Spec.Replicas = &replicas
		_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, st, metav1.UpdateOptions{})
		if err != nil {
			if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
				return common.NewConflict("k8s.workload.resource_version_conflict")
			}
			return errors.New("k8s.workload.scale_failed")
		}
		return nil
	default:
		d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.New("k8s.workload.not_found")
		}
		if expectedResourceVersion != "" && d.ResourceVersion != expectedResourceVersion {
			return common.NewConflict("k8s.workload.resource_version_conflict")
		}
		d.Spec.Replicas = &replicas
		_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, d, metav1.UpdateOptions{})
		if err != nil {
			if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
				return common.NewConflict("k8s.workload.resource_version_conflict")
			}
			return errors.New("k8s.workload.scale_failed")
		}
		return nil
	}
}

// Restart triggers a rolling restart by patching the pod template annotation.
func (s *WorkloadService) Restart(clusterID uint64, namespace, kind, name string, dataScope *common.DataScopeReq) error {
	return s.RestartWithResourceVersion(clusterID, namespace, kind, name, "", dataScope)
}

func (s *WorkloadService) RestartWithResourceVersion(clusterID uint64, namespace, kind, name, expectedResourceVersion string, dataScope *common.DataScopeReq) error {
	if s.namespaceSvc == nil {
		return errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "workload:restart"); err != nil {
		return err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().UTC().Format(time.RFC3339))

	switch normalizeKind(kind) {
	case workloadKindStatefulSet:
		st, getErr := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return errors.New("k8s.workload.not_found")
		}
		if expectedResourceVersion != "" && st.ResourceVersion != expectedResourceVersion {
			return common.NewConflict("k8s.workload.resource_version_conflict")
		}
		_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	default:
		d, getErr := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return errors.New("k8s.workload.not_found")
		}
		if expectedResourceVersion != "" && d.ResourceVersion != expectedResourceVersion {
			return common.NewConflict("k8s.workload.resource_version_conflict")
		}
		_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	}
	if err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict("k8s.workload.resource_version_conflict")
		}
		return errors.New("k8s.workload.restart_failed")
	}
	return nil
}

func (s *WorkloadService) workloadSelector(ctx context.Context, clientset kubernetes.Interface, namespace, kind, name string) (string, error) {
	switch normalizeKind(kind) {
	case workloadKindStatefulSet:
		st, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", errors.New("k8s.workload.not_found")
		}
		return metav1.FormatLabelSelector(st.Spec.Selector), nil
	case workloadKindDaemonSet:
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", errors.New("k8s.workload.not_found")
		}
		return metav1.FormatLabelSelector(ds.Spec.Selector), nil
	default:
		d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", errors.New("k8s.workload.not_found")
		}
		return metav1.FormatLabelSelector(d.Spec.Selector), nil
	}
}

func normalizeKind(kind string) string {
	switch kind {
	case workloadKindDeployment, "Deployment":
		return workloadKindDeployment
	case workloadKindStatefulSet, "StatefulSet":
		return workloadKindStatefulSet
	case workloadKindDaemonSet, "DaemonSet":
		return workloadKindDaemonSet
	default:
		return ""
	}
}

func containerImages(containers []corev1.Container) []string {
	images := make([]string, 0, len(containers))
	for _, c := range containers {
		images = append(images, c.Image)
	}
	return images
}

func safeReplicas(r *int32) int32 {
	if r == nil {
		return 0
	}
	return *r
}

func workloadStatus(ready, desired int32) string {
	if desired == 0 {
		return "scaled_down"
	}
	if ready >= desired {
		return "ready"
	}
	return "progressing"
}

func podRestarts(p *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range p.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

func age(t metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t.Time)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
