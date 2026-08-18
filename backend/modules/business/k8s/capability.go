package k8s

import (
	"context"
	"errors"
	"strings"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type targetReader struct {
	clusters interface {
		GetClientset(uint64, *common.DataScopeReq) (*kubernetes.Clientset, error)
		GetMeta(uint64, *common.DataScopeReq) (uint64, uint64, error)
	}
}

func NewTargetReader(clusters *cluster.ClusterService) bizcap.K8sTargetReader {
	return &targetReader{clusters: clusters}
}

func (r *targetReader) ResolveTarget(ctx context.Context, req bizcap.K8sTargetQuery) (bizcap.K8sTargetRef, error) {
	if req.ClusterID == 0 || strings.TrimSpace(req.Namespace) == "" || strings.TrimSpace(req.WorkloadName) == "" {
		return bizcap.K8sTargetRef{}, errors.New("business.service.target_invalid")
	}
	client, err := r.clusters.GetClientset(req.ClusterID, req.DataScope)
	if err != nil {
		return bizcap.K8sTargetRef{}, err
	}
	kind := normalizeWorkloadKind(req.WorkloadKind)
	switch kind {
	case "Deployment":
		if _, err = client.AppsV1().Deployments(req.Namespace).Get(ctx, req.WorkloadName, metav1.GetOptions{}); err != nil {
			return bizcap.K8sTargetRef{}, errors.New("business.service.k8s_target_not_found")
		}
	case "StatefulSet":
		if _, err = client.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.WorkloadName, metav1.GetOptions{}); err != nil {
			return bizcap.K8sTargetRef{}, errors.New("business.service.k8s_target_not_found")
		}
	case "DaemonSet":
		if _, err = client.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.WorkloadName, metav1.GetOptions{}); err != nil {
			return bizcap.K8sTargetRef{}, errors.New("business.service.k8s_target_not_found")
		}
	default:
		return bizcap.K8sTargetRef{}, errors.New("business.service.target_invalid")
	}
	scopeID, deptID, err := r.clusters.GetMeta(req.ClusterID, req.DataScope)
	if err != nil {
		return bizcap.K8sTargetRef{}, err
	}
	return bizcap.K8sTargetRef{ClusterID: req.ClusterID, Namespace: req.Namespace, WorkloadKind: kind, WorkloadName: req.WorkloadName, BusinessScopeID: scopeID, DeptID: deptID}, nil
}

func normalizeWorkloadKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment", "deployments":
		return "Deployment"
	case "statefulset", "statefulsets":
		return "StatefulSet"
	case "daemonset", "daemonsets":
		return "DaemonSet"
	default:
		return ""
	}
}
