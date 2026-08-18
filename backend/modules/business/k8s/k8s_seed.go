package k8s

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type menuSeed struct {
	Key        string
	ParentPath string
	ParentKey  string
	TitleKey   string
	Path       string
	Component  string
	PagePerm   string
	Perms      string
	Type       string
	Icon       string
	RouteName  string
	Module     string
	Sort       int
}

var seeds = []menuSeed{
	{
		Key:       "k8s",
		TitleKey:  businessK8sMenu,
		Path:      "/business/k8s",
		Type:      "M",
		Icon:      "cloud",
		RouteName: "k8s",
		Module:    k8sModuleKey,
		Sort:      4,
	},
	{
		Key:       "k8s-cluster",
		ParentKey: "k8s",
		TitleKey:  operationsK8sClusterMenu,
		Path:      "/business/k8s/cluster",
		Component: "business/k8s/cluster/ClusterList",
		PagePerm:  "business:k8s:cluster:list",
		Type:      "C",
		Icon:      "apps",
		RouteName: "k8s-cluster-list",
		Module:    k8sModuleKey,
		Sort:      3,
	},
	{
		Key:       "k8s-cluster-view",
		ParentKey: "k8s-cluster",
		TitleKey:  k8sClusterPermViewKey,
		Perms:     "business:k8s:cluster:view",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      1,
	},
	{
		Key:       "k8s-cluster-create",
		ParentKey: "k8s-cluster",
		TitleKey:  k8sClusterPermCreateKey,
		Perms:     "business:k8s:cluster:create",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      2,
	},
	{
		Key:       "k8s-cluster-update",
		ParentKey: "k8s-cluster",
		TitleKey:  k8sClusterPermUpdateKey,
		Perms:     "business:k8s:cluster:update",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      3,
	},
	{
		Key:       "k8s-cluster-delete",
		ParentKey: "k8s-cluster",
		TitleKey:  k8sClusterPermDeleteKey,
		Perms:     "business:k8s:cluster:delete",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      4,
	},
	{
		Key:       "k8s-workload",
		ParentKey: "k8s",
		TitleKey:  operationsK8sWorkloadMenu,
		Path:      "/business/k8s/workload",
		Component: "business/k8s/workload/WorkloadList",
		PagePerm:  "business:k8s:workload:list",
		Type:      "C",
		Icon:      "storage",
		RouteName: "k8s-workload-list",
		Module:    k8sModuleKey,
		Sort:      4,
	},
	{
		Key:       "k8s-workload-view",
		ParentKey: "k8s-workload",
		TitleKey:  k8sWorkloadPermViewKey,
		Perms:     "business:k8s:workload:view",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      1,
	},
	{
		Key:       "k8s-workload-update",
		ParentKey: "k8s-workload",
		TitleKey:  k8sWorkloadPermUpdateKey,
		Perms:     "business:k8s:workload:update",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      2,
	},
	{
		Key:       "k8s-release",
		ParentKey: "k8s",
		TitleKey:  operationsK8sReleaseMenu,
		Path:      "/business/k8s/release",
		Component: "business/k8s/release/ReleaseList",
		PagePerm:  "business:k8s:release:list",
		Type:      "C",
		Icon:      "code",
		RouteName: "k8s-release-list",
		Module:    k8sModuleKey,
		Sort:      5,
	},
	{
		Key:       "k8s-release-view",
		ParentKey: "k8s-release",
		TitleKey:  k8sReleasePermViewKey,
		Perms:     "business:k8s:release:view",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      1,
	},
	{
		Key:       "k8s-release-create",
		ParentKey: "k8s-release",
		TitleKey:  k8sReleasePermCreateKey,
		Perms:     "business:k8s:release:create",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      2,
	},
	{
		Key:       "k8s-release-rollback",
		ParentKey: "k8s-release",
		TitleKey:  k8sReleasePermRollbackKey,
		Perms:     "business:k8s:release:rollback",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      3,
	},
	{
		Key:       "k8s-release-reconcile",
		ParentKey: "k8s-release",
		TitleKey:  k8sReleasePermReconcileKey,
		Perms:     "business:k8s:release:reconcile",
		Type:      "F",
		Module:    k8sModuleKey,
		Sort:      4,
	},
}

type i18nSeed struct {
	Module string
	Locale string
	Group  string
	Key    string
	Value  string
}

var i18nSeeds = []i18nSeed{
	// K8s module container
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "menu", Key: businessK8sMenu, Value: "Kubernetes"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "menu", Key: businessK8sMenu, Value: "Kubernetes"},
	// Cluster menu & detail
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "menu", Key: operationsK8sClusterMenu, Value: "K8s 集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "menu", Key: operationsK8sClusterMenu, Value: "K8s Clusters"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.detail", Value: "集群详情"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.detail", Value: "Cluster Detail"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "menu", Key: operationsK8sWorkloadMenu, Value: "工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "menu", Key: operationsK8sWorkloadMenu, Value: "Workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "menu", Key: operationsK8sReleaseMenu, Value: "应用发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "menu", Key: operationsK8sReleaseMenu, Value: "Releases"},

	// Cluster hero & fields
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.hero.eyebrow", Value: "运维平台 / K8s 集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.hero.eyebrow", Value: "Operations / K8s Clusters"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.hero.title", Value: "纳管多集群，统一管理容器资源"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.hero.title", Value: "Manage multiple clusters and container resources"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.hero.total", Value: "集群总数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.hero.total", Value: "Total Clusters"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.hero.healthy", Value: "健康集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.hero.healthy", Value: "Healthy"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.hero.nodes", Value: "节点总数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.hero.nodes", Value: "Nodes"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.code", Value: "集群编码"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.code", Value: "Cluster Code"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.codePlaceholder", Value: "请输入集群编码"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.codePlaceholder", Value: "Enter cluster code"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.name", Value: "集群名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.name", Value: "Cluster Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.namePlaceholder", Value: "请输入集群名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.namePlaceholder", Value: "Enter cluster name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.environment", Value: "环境"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.environment", Value: "Environment"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.environment.dev", Value: "开发"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.environment.dev", Value: "Development"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.environment.test", Value: "测试"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.environment.test", Value: "Testing"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.environment.prod", Value: "生产"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.environment.prod", Value: "Production"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.businessScope", Value: "业务域"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.businessScope", Value: "Business Scope"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.businessScopePlaceholder", Value: "请选择业务域"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.businessScopePlaceholder", Value: "Select business scope"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.version", Value: "版本"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.version", Value: "Version"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.nodes", Value: "节点"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.nodes", Value: "Nodes"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.pods", Value: "Pod"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.pods", Value: "Pods"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.status", Value: "状态"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.status", Value: "Status"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.status.healthy", Value: "健康"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.status.healthy", Value: "Healthy"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.status.unreachable", Value: "不可达"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.status.unreachable", Value: "Unreachable"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.status.unknown", Value: "未同步"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.status.unknown", Value: "Unknown"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.status.degraded", Value: "降级"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.status.degraded", Value: "Degraded"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.lastSyncedAt", Value: "最后同步时间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.lastSyncedAt", Value: "Last Synced"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.apiServer", Value: "API Server"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.apiServer", Value: "API Server"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.cpu", Value: "CPU"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.cpu", Value: "CPU"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.memory", Value: "内存"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.memory", Value: "Memory"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.kubeconfig", Value: "Kubeconfig"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.kubeconfig", Value: "Kubeconfig"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.kubeconfigPlaceholder", Value: "请粘贴 Kubeconfig 内容"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.kubeconfigPlaceholder", Value: "Paste kubeconfig content"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.kubeconfigUpdateHint", Value: "留空表示不修改"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.kubeconfigUpdateHint", Value: "Leave empty to keep unchanged"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.remark", Value: "备注"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.remark", Value: "Remark"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.field.keywordPlaceholder", Value: "搜索集群编码或名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.field.keywordPlaceholder", Value: "Search cluster code or name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.createTitle", Value: "注册集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.createTitle", Value: "Register Cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.editTitle", Value: "编辑集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.editTitle", Value: "Edit Cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.deleteConfirm", Value: "确认删除该集群？"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.deleteConfirm", Value: "Delete this cluster?"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.empty", Value: "暂无集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.empty", Value: "No clusters"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.sync", Value: "同步"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.sync", Value: "Sync"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.syncSuccess", Value: "集群同步成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.syncSuccess", Value: "Cluster synced"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.syncFailed", Value: "集群同步失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.syncFailed", Value: "Cluster sync failed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.summaryTitle", Value: "集群概览"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.summaryTitle", Value: "Cluster Overview"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.nodesTab", Value: "节点"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.nodesTab", Value: "Nodes"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.namespacesTab", Value: "命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.namespacesTab", Value: "Namespaces"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.name", Value: "节点名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.name", Value: "Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.status", Value: "状态"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.status", Value: "Status"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.status.ready", Value: "就绪"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.status.ready", Value: "Ready"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.status.not_ready", Value: "未就绪"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.status.not_ready", Value: "Not Ready"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.ip", Value: "IP"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.ip", Value: "IP"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.os", Value: "操作系统"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.os", Value: "OS"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.kubelet", Value: "Kubelet"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.kubelet", Value: "Kubelet"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.cpu", Value: "CPU"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.cpu", Value: "CPU"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.memory", Value: "内存"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.memory", Value: "Memory"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.node.pods", Value: "Pod 数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.node.pods", Value: "Pods"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.nodesLoadFailed", Value: "节点列表加载失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.nodesLoadFailed", Value: "Failed to load nodes"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.nodesEmpty", Value: "暂无节点"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.nodesEmpty", Value: "No nodes"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.namespacesLoadFailed", Value: "命名空间列表加载失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.namespacesLoadFailed", Value: "Failed to load namespaces"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.namespacesEmpty", Value: "暂无命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.namespacesEmpty", Value: "No namespaces"},

	// Workload
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.hero.eyebrow", Value: "运维平台 / 工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.hero.eyebrow", Value: "Operations / Workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.hero.title", Value: "查看和管理 Deployment、StatefulSet 与 DaemonSet"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.hero.title", Value: "View and manage Deployments, StatefulSets and DaemonSets"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.hero.total", Value: "工作负载总数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.hero.total", Value: "Total Workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.hero.ready", Value: "就绪"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.hero.ready", Value: "Ready"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.hero.progressing", Value: "更新中"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.hero.progressing", Value: "Progressing"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.cluster", Value: "集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.cluster", Value: "Cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.kind", Value: "类型"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.kind", Value: "Kind"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.kind.deployment", Value: "Deployment"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.kind.deployment", Value: "Deployment"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.kind.statefulset", Value: "StatefulSet"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.kind.statefulset", Value: "StatefulSet"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.kind.daemonset", Value: "DaemonSet"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.kind.daemonset", Value: "DaemonSet"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.name", Value: "名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.name", Value: "Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.namespace", Value: "命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.namespace", Value: "Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.namespacePlaceholder", Value: "输入命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.namespacePlaceholder", Value: "Enter namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.replicas", Value: "副本数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.replicas", Value: "Replicas"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.images", Value: "镜像"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.images", Value: "Images"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.status", Value: "状态"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.status", Value: "Status"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.status.ready", Value: "就绪"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.status.ready", Value: "Ready"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.status.progressing", Value: "更新中"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.status.progressing", Value: "Progressing"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.status.scaled_down", Value: "已缩容"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.status.scaled_down", Value: "Scaled Down"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.field.age", Value: "时长"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.field.age", Value: "Age"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.scale", Value: "扩缩容"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.scale", Value: "Scale"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.scaleTitle", Value: "扩缩容 {{name}}"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.scaleTitle", Value: "Scale {{name}}"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.scaleSuccess", Value: "扩缩容成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.scaleSuccess", Value: "Scaled successfully"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.restart", Value: "重启"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.restart", Value: "Restart"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.restartConfirm", Value: "确认重启该工作负载？"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.restartConfirm", Value: "Restart this workload?"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.restartSuccess", Value: "重启成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.restartSuccess", Value: "Restarted successfully"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.restartFailed", Value: "重启失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.restartFailed", Value: "Restart failed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pods", Value: "Pod"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pods", Value: "Pods"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.podsTitle", Value: "Pod 列表 - {{name}}"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.podsTitle", Value: "Pods - {{name}}"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.podsEmpty", Value: "暂无 Pod"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.podsEmpty", Value: "No pods"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pod.name", Value: "Pod 名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pod.name", Value: "Pod Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pod.status", Value: "状态"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pod.status", Value: "Status"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pod.node", Value: "节点"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pod.node", Value: "Node"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pod.restarts", Value: "重启次数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pod.restarts", Value: "Restarts"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.pod.createdAt", Value: "创建时间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.pod.createdAt", Value: "Created"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.selectClusterHint", Value: "请先选择集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.selectClusterHint", Value: "Select a cluster first"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.empty", Value: "暂无工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.empty", Value: "No workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.view", Value: "日志"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.view", Value: "Logs"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.title", Value: "日志 - {{name}}"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.title", Value: "Logs - {{name}}"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.connecting", Value: "连接中"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.connecting", Value: "Connecting"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.connected", Value: "已连接"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.connected", Value: "Connected"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.closed", Value: "已断开"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.closed", Value: "Closed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.error", Value: "连接错误"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.error", Value: "Error"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.reconnect", Value: "重连"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.reconnect", Value: "Reconnect"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.clear", Value: "清空"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.clear", Value: "Clear"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.workload.log.loading", Value: "正在加载日志..."},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.workload.log.loading", Value: "Loading logs..."},

	// Release
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.hero.eyebrow", Value: "运维平台 / 应用发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.hero.eyebrow", Value: "Operations / Releases"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.hero.title", Value: "管理应用镜像更新与回滚"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.hero.title", Value: "Manage application image updates and rollbacks"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.hero.total", Value: "发布总数"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.hero.total", Value: "Total Releases"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.hero.success", Value: "成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.hero.success", Value: "Succeeded"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.hero.failed", Value: "失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.hero.failed", Value: "Failed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.name", Value: "发布名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.name", Value: "Release Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.namePlaceholder", Value: "请输入发布名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.namePlaceholder", Value: "Enter release name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.cluster", Value: "集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.cluster", Value: "Cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.namespace", Value: "命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.namespace", Value: "Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.namespacePlaceholder", Value: "输入命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.namespacePlaceholder", Value: "Enter namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.workloadType", Value: "工作负载类型"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.workloadType", Value: "Workload Type"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.workloadName", Value: "工作负载名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.workloadName", Value: "Workload Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.workloadNamePlaceholder", Value: "输入工作负载名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.workloadNamePlaceholder", Value: "Enter workload name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.containerName", Value: "容器名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.containerName", Value: "Container Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.containerNamePlaceholder", Value: "默认第一个容器"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.containerNamePlaceholder", Value: "Defaults to first container"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.image", Value: "目标镜像"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.image", Value: "Target Image"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.imagePlaceholder", Value: "如 nginx:1.25"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.imagePlaceholder", Value: "e.g. nginx:1.25"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.imageBefore", Value: "发布前镜像"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.imageBefore", Value: "Image Before"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.imageAfter", Value: "发布后镜像"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.imageAfter", Value: "Image After"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.workload", Value: "工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.workload", Value: "Workload"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.status", Value: "状态"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.status", Value: "Status"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.rollout", Value: "就绪副本"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.rollout", Value: "Ready Replicas"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.condition", Value: "Rollout 观测"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.condition", Value: "Rollout Observation"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.success", Value: "成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.success", Value: "Succeeded"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.succeeded", Value: "成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.succeeded", Value: "Succeeded"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.failed", Value: "失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.failed", Value: "Failed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.pending", Value: "进行中"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.pending", Value: "Pending"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.applying", Value: "应用中"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.applying", Value: "Applying"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.timed_out", Value: "观测超时"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.timed_out", Value: "Observation Timed Out"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.status.rollback_success", Value: "回滚成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.status.rollback_success", Value: "Rolled Back"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.field.createdAt", Value: "发布时间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.field.createdAt", Value: "Created"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.create", Value: "创建发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.create", Value: "Create Release"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.createTitle", Value: "创建发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.createTitle", Value: "Create Release"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.rollback", Value: "回滚"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.rollback", Value: "Rollback"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.rollbackConfirm", Value: "确认回滚到上一个版本？"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.rollbackConfirm", Value: "Roll back to the previous version?"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.rollbackSuccess", Value: "回滚成功"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.rollbackSuccess", Value: "Rolled back successfully"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.reconcile", Value: "重新观测"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.reconcile", Value: "Reconcile"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.reconcileSuccess", Value: "发布状态已重新观测"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.reconcileSuccess", Value: "Release state reconciled"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.release.empty", Value: "暂无发布记录"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.release.empty", Value: "No releases"},

	// Permissions
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sClusterPermViewKey, Value: "查看集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sClusterPermViewKey, Value: "View cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sClusterPermCreateKey, Value: "注册集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sClusterPermCreateKey, Value: "Create cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sClusterPermUpdateKey, Value: "编辑集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sClusterPermUpdateKey, Value: "Update cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sClusterPermDeleteKey, Value: "删除集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sClusterPermDeleteKey, Value: "Delete cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sWorkloadPermViewKey, Value: "查看工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sWorkloadPermViewKey, Value: "View workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sWorkloadPermUpdateKey, Value: "管理工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sWorkloadPermUpdateKey, Value: "Manage workloads"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sReleasePermViewKey, Value: "查看发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sReleasePermViewKey, Value: "View releases"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sReleasePermCreateKey, Value: "创建发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sReleasePermCreateKey, Value: "Create release"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sReleasePermRollbackKey, Value: "回滚发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sReleasePermRollbackKey, Value: "Rollback release"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "permission", Key: k8sReleasePermReconcileKey, Value: "重新观测发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "permission", Key: k8sReleasePermReconcileKey, Value: "Reconcile release"},

	// Audit
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.cluster.audit.create", Value: "注册集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.cluster.audit.create", Value: "Create cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.cluster.audit.update", Value: "编辑集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.cluster.audit.update", Value: "Update cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.cluster.audit.delete", Value: "删除集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.cluster.audit.delete", Value: "Delete cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.cluster.audit.sync", Value: "同步集群"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.cluster.audit.sync", Value: "Sync cluster"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.workload.audit.scale", Value: "工作负载扩缩容"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.workload.audit.scale", Value: "Scale workload"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.workload.audit.restart", Value: "重启工作负载"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.workload.audit.restart", Value: "Restart workload"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.release.audit.create", Value: "创建发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.release.audit.create", Value: "Create release"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "audit", Key: "k8s.release.audit.rollback", Value: "回滚发布"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "audit", Key: "k8s.release.audit.rollback", Value: "Rollback release"},

	// Cluster tabs & namespace
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.configmapsTab", Value: "ConfigMap"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.configmapsTab", Value: "ConfigMaps"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.cluster.secretsTab", Value: "Secret"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.cluster.secretsTab", Value: "Secrets"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.namespace.create", Value: "创建命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.namespace.create", Value: "Create Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.namespace.createTitle", Value: "创建命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.namespace.createTitle", Value: "Create Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.namespace.namePlaceholder", Value: "请输入命名空间名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.namespace.namePlaceholder", Value: "Enter namespace name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.namespace.deleteConfirm", Value: "确认删除命名空间 {{name}}？该操作会删除命名空间下所有资源。"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.namespace.deleteConfirm", Value: "Delete namespace {{name}}? This removes all resources in it."},

	// ConfigMap
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.name", Value: "名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.name", Value: "Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.namespace", Value: "命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.namespace", Value: "Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.keyCount", Value: "键数量"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.keyCount", Value: "Keys"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.create", Value: "创建 ConfigMap"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.create", Value: "Create ConfigMap"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.createTitle", Value: "创建 ConfigMap"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.createTitle", Value: "Create ConfigMap"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.namePlaceholder", Value: "请输入 ConfigMap 名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.namePlaceholder", Value: "Enter ConfigMap name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.namespacePlaceholder", Value: "请输入命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.namespacePlaceholder", Value: "Enter namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.data", Value: "数据"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.data", Value: "Data"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.dataHint", Value: "每行一个键值对，格式 key=value"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.dataHint", Value: "One key=value pair per line"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.dataPlaceholder", Value: "key1=value1\nkey2=value2"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.dataPlaceholder", Value: "key1=value1\nkey2=value2"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.deleteConfirm", Value: "确认删除 ConfigMap {{name}}？"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.deleteConfirm", Value: "Delete ConfigMap {{name}}?"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.loadFailed", Value: "ConfigMap 列表加载失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.loadFailed", Value: "Failed to load ConfigMaps"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.configmap.empty", Value: "暂无 ConfigMap"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.configmap.empty", Value: "No ConfigMaps"},

	// Secret
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.name", Value: "名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.name", Value: "Name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.namespace", Value: "命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.namespace", Value: "Namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.type", Value: "类型"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.type", Value: "Type"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.keyCount", Value: "键数量"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.keyCount", Value: "Keys"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.create", Value: "创建 Secret"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.create", Value: "Create Secret"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.createTitle", Value: "创建 Secret"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.createTitle", Value: "Create Secret"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.namePlaceholder", Value: "请输入 Secret 名称"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.namePlaceholder", Value: "Enter Secret name"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.namespacePlaceholder", Value: "请输入命名空间"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.namespacePlaceholder", Value: "Enter namespace"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.data", Value: "数据"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.data", Value: "Data"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.dataHint", Value: "每行一个键值对，格式 key=value（值将加密存储于集群）"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.dataHint", Value: "One key=value pair per line (values stored encrypted in cluster)"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.dataPlaceholder", Value: "password=secret123"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.dataPlaceholder", Value: "password=secret123"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.deleteConfirm", Value: "确认删除 Secret {{name}}？"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.deleteConfirm", Value: "Delete Secret {{name}}?"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.loadFailed", Value: "Secret 列表加载失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.loadFailed", Value: "Failed to load Secrets"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "page", Key: "business.k8s.secret.empty", Value: "暂无 Secret"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "page", Key: "business.k8s.secret.empty", Value: "No Secrets"},

	// Errors
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.cluster.code_exists", Value: "集群编码已存在"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.cluster.code_exists", Value: "Cluster code already exists"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.cluster.not_found", Value: "集群不存在"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.cluster.not_found", Value: "Cluster does not exist"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.cluster.sync_failed", Value: "集群同步失败"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.cluster.sync_failed", Value: "Cluster sync failed"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.kubeconfig_invalid", Value: "Kubeconfig 无效"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.kubeconfig_invalid", Value: "Invalid kubeconfig"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.kubeconfig_key_missing", Value: "未配置 K8s 加密密钥"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.kubeconfig_key_missing", Value: "K8s encryption key not configured"},
	{Module: k8sModuleKey, Locale: "zh-CN", Group: "error", Key: "k8s.workload.not_found", Value: "工作负载不存在"},
	{Module: k8sModuleKey, Locale: "en-US", Group: "error", Key: "k8s.workload.not_found", Value: "Workload does not exist"},
}

func seedMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		keyToID := make(map[string]uint64, len(seeds))
		for _, seed := range seeds {
			if _, err := ensureMenuSeed(tx, keyToID, seed); err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureMenuSeed(tx *gorm.DB, keyToID map[string]uint64, seed menuSeed) (uint64, error) {
	menuID, err := findMenuSeedID(tx, seed)
	if err != nil {
		return 0, err
	}

	parentID, err := resolveMenuSeedParentID(tx, keyToID, seed)
	if err != nil {
		return 0, err
	}

	payload := map[string]interface{}{
		"parent_id":  parentID,
		"title_key":  seed.TitleKey,
		"path":       seed.Path,
		"component":  seed.Component,
		"page_perm":  seed.PagePerm,
		"perms":      seed.Perms,
		"type":       seed.Type,
		"icon":       seed.Icon,
		"route_name": seed.RouteName,
		"module":     seed.Module,
		"sort":       seed.Sort,
		"is_visible": 1,
		"is_cache":   0,
		"updated_at": time.Now(),
	}

	if menuID == 0 {
		payload["created_at"] = time.Now()
		if err := tx.Table("system_menu").Create(payload).Error; err != nil {
			return 0, err
		}
		if menuID, err = findMenuSeedID(tx, seed); err != nil {
			return 0, err
		}
	} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
		return 0, err
	}

	if seed.Key != "" {
		keyToID[seed.Key] = menuID
	}
	if err := bindAdmin(tx, menuID, seed); err != nil {
		return 0, err
	}
	return menuID, nil
}

func findMenuSeedID(tx *gorm.DB, seed menuSeed) (uint64, error) {
	var menuID uint64
	if seed.Path != "" {
		if err := tx.Table("system_menu").Select("id").Where(menuPathWhereClause, seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	} else if seed.Perms != "" {
		if err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	}
	return menuID, nil
}

func resolveMenuSeedParentID(tx *gorm.DB, keyToID map[string]uint64, seed menuSeed) (uint64, error) {
	parentID := uint64(0)
	if seed.ParentKey != "" {
		parentID = keyToID[seed.ParentKey]
	}
	if parentID == 0 && seed.ParentPath != "" {
		if err := tx.Table("system_menu").Select("id").Where(menuPathWhereClause, seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
			return 0, err
		}
	}
	return parentID, nil
}

func bindAdmin(tx *gorm.DB, menuID uint64, seed menuSeed) error {
	if menuID == 0 || !tx.Migrator().HasTable("system_role") {
		return nil
	}
	var adminRoleID uint64
	if err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}
	if err := bindAdminMenu(tx, adminRoleID, menuID, seed); err != nil {
		return err
	}
	return bindAdminPermission(tx, adminRoleID, seed)
}

func bindAdminMenu(tx *gorm.DB, adminRoleID, menuID uint64, seed menuSeed) error {
	if seed.Type != "C" || !tx.Migrator().HasTable("system_role_menu") {
		return nil
	}
	var count int64
	if err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	return tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error
}

func bindAdminPermission(tx *gorm.DB, adminRoleID uint64, seed menuSeed) error {
	if !tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
	for _, permissionKey := range []string{strings.TrimSpace(seed.PagePerm), strings.TrimSpace(seed.Perms)} {
		if permissionKey == "" {
			continue
		}
		var count int64
		if err := tx.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range i18nSeeds {
			var count int64
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Count(&count).Error; err != nil {
				return err
			}
			payload := map[string]interface{}{
				"module":           seed.Module,
				"group_name":       seed.Group,
				"key":              seed.Key,
				"locale":           seed.Locale,
				"value":            seed.Value,
				"lifecycle_status": "active",
				"updated_at":       time.Now(),
			}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
