package backup

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const veleroNamespace = "velero"

// VeleroClient 封装对 K8s 集群中 Velero 的访问。
// Velero 通过 CRD（backups.velero.io、schedules.velero.io、restores.velero.io、
// backupstoragelocations.velero.io）工作。CRD 的实际创建/查询依赖 dynamic client，
// 这里先提供安装检测能力；CRD 操作通过后续扩展 cluster service 的 dynamic client 接入。
type VeleroClient struct {
	clientset kubernetes.Interface
}

// NewVeleroClient 创建 Velero 客户端
func NewVeleroClient(clientset kubernetes.Interface) *VeleroClient {
	return &VeleroClient{clientset: clientset}
}

// IsInstalled 检查 Velero 是否已安装（通过检测 velero namespace 是否存在）
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

// GetVersion 获取 Velero 版本（从 velero namespace 的 deployment 标签读取）
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
			// 版本通常在镜像 tag 中
			image := d.Spec.Template.Spec.Containers[0].Image
			return image, nil
		}
	}
	return "", nil
}

// GetRunningPods 获取 velero namespace 中运行中的 Pod 数量
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

// NOTE: 以下 CRD 操作方法（CreateBackup、CreateSchedule、CreateRestore、
// ListBackups 等）需要 dynamic client 来操作 velero.io/v1 资源。
// 当前 cluster service 仅暴露 *kubernetes.Clientset，未暴露 dynamic.Interface。
// 待 cluster service 增加 GetDynamicClient 方法后，在此补充完整的 CRD 操作实现。
//
// 数据库层（biz_velero_backup 等表）已完成完整的 CRUD，可在 API 层闭环；
// 真正的 Velero K8s 资源创建需要通过后续集成。
