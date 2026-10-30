package workload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	syncTimeout                = 30 * time.Second
	workloadKindDeployment     = "deployment"
	workloadKindStatefulSet    = "statefulset"
	workloadKindDaemonSet      = "daemonset"
	msgResourceVersionConflict = "k8s.workload.resource_version_conflict"
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

	options := metav1.ListOptions{Limit: normalizeListLimit(query.Limit), Continue: query.ContinueToken}
	var items []WorkloadItem
	var continueToken string
	switch normalizeKind(query.Kind) {
	case workloadKindDeployment:
		items, continueToken, err = listDeployments(ctx, clientset, ns, options)
	case workloadKindStatefulSet:
		items, continueToken, err = listStatefulSets(ctx, clientset, ns, options)
	case workloadKindDaemonSet:
		items, continueToken, err = listDaemonSets(ctx, clientset, ns, options)
	default:
		items, err = listAllWorkloadKinds(ctx, clientset, ns, options)
	}
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(query.Kind) == "" {
		// Multi-kind lists cannot safely compose Kubernetes continuation tokens;
		// callers must select a kind to continue beyond this bounded first page.
		continueToken = ""
	}
	return &WorkloadListResponse{Items: items, Total: len(items), ContinueToken: continueToken}, nil
}

// listDeployments reads one deployment page and maps it to workload items with
// the continuation token for the next page.
func listDeployments(ctx context.Context, clientset kubernetes.Interface, ns string, options metav1.ListOptions) ([]WorkloadItem, string, error) {
	list, err := clientset.AppsV1().Deployments(ns).List(ctx, options)
	if err != nil {
		return nil, "", errors.New("k8s.workload.list_failed")
	}
	items := make([]WorkloadItem, 0, len(list.Items))
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
	return items, list.Continue, nil
}

// listStatefulSets reads one statefulset page and maps it to workload items.
func listStatefulSets(ctx context.Context, clientset kubernetes.Interface, ns string, options metav1.ListOptions) ([]WorkloadItem, string, error) {
	list, err := clientset.AppsV1().StatefulSets(ns).List(ctx, options)
	if err != nil {
		return nil, "", errors.New("k8s.workload.list_failed")
	}
	items := make([]WorkloadItem, 0, len(list.Items))
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
	return items, list.Continue, nil
}

// listDaemonSets reads one daemonset page and maps it to workload items.
func listDaemonSets(ctx context.Context, clientset kubernetes.Interface, ns string, options metav1.ListOptions) ([]WorkloadItem, string, error) {
	list, err := clientset.AppsV1().DaemonSets(ns).List(ctx, options)
	if err != nil {
		return nil, "", errors.New("k8s.workload.list_failed")
	}
	items := make([]WorkloadItem, 0, len(list.Items))
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
	return items, list.Continue, nil
}

// listAllWorkloadKinds aggregates the three kinds; only the deployment page
// carries the caller's limit and continuation token, as before.
func listAllWorkloadKinds(ctx context.Context, clientset kubernetes.Interface, ns string, options metav1.ListOptions) ([]WorkloadItem, error) {
	deployments, _, err := listDeployments(ctx, clientset, ns, options)
	if err != nil {
		return nil, err
	}
	statefulsets, _, err := listStatefulSets(ctx, clientset, ns, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	daemonsets, _, err := listDaemonSets(ctx, clientset, ns, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]WorkloadItem, 0, len(deployments)+len(statefulsets)+len(daemonsets))
	items = append(items, deployments...)
	items = append(items, statefulsets...)
	items = append(items, daemonsets...)
	return items, nil
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

// ScaleWithResourceVersion scales a workload when its resource version matches.
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

	if normalizeKind(kind) == workloadKindStatefulSet {
		return scaleStatefulSet(ctx, clientset, namespace, name, replicas, expectedResourceVersion)
	}
	return scaleDeployment(ctx, clientset, namespace, name, replicas, expectedResourceVersion)
}

// scaleDeployment patches the replica count of a deployment, refusing a stale
// resource version both before the write and on an API-server conflict.
func scaleDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, name string, replicas int32, expectedResourceVersion string) error {
	d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.New("k8s.workload.not_found")
	}
	if isResourceVersionMismatch(d.ResourceVersion, expectedResourceVersion) {
		return common.NewConflict(msgResourceVersionConflict)
	}
	d.Spec.Replicas = &replicas
	if _, err := clientset.AppsV1().Deployments(namespace).Update(ctx, d, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict(msgResourceVersionConflict)
		}
		return errors.New("k8s.workload.scale_failed")
	}
	return nil
}

// scaleStatefulSet patches the replica count of a statefulset, refusing a stale
// resource version both before the write and on an API-server conflict.
func scaleStatefulSet(ctx context.Context, clientset kubernetes.Interface, namespace, name string, replicas int32, expectedResourceVersion string) error {
	st, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.New("k8s.workload.not_found")
	}
	if isResourceVersionMismatch(st.ResourceVersion, expectedResourceVersion) {
		return common.NewConflict(msgResourceVersionConflict)
	}
	st.Spec.Replicas = &replicas
	if _, err := clientset.AppsV1().StatefulSets(namespace).Update(ctx, st, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict(msgResourceVersionConflict)
		}
		return errors.New("k8s.workload.scale_failed")
	}
	return nil
}

// isResourceVersionMismatch reports a caller-supplied resource version that no
// longer matches the object read from the cluster; an empty expectation skips it.
func isResourceVersionMismatch(actual, expected string) bool {
	return expected != "" && actual != expected
}

// Restart triggers a rolling restart by patching the pod template annotation.
func (s *WorkloadService) Restart(clusterID uint64, namespace, kind, name string, dataScope *common.DataScopeReq) error {
	return s.RestartWithResourceVersion(clusterID, namespace, kind, name, "", dataScope)
}

// RestartWithResourceVersion restarts a workload when its resource version matches.
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
			return common.NewConflict(msgResourceVersionConflict)
		}
		_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	default:
		d, getErr := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return errors.New("k8s.workload.not_found")
		}
		if expectedResourceVersion != "" && d.ResourceVersion != expectedResourceVersion {
			return common.NewConflict(msgResourceVersionConflict)
		}
		_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	}
	if err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict(msgResourceVersionConflict)
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
