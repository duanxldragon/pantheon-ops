package k8s

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// registerK8sPermissionRoutes declares the K8s action bindings used by scoped
// roles. Only assigned keys grant policies; a read key never includes mutation
// verbs. Routes stay unregistered where no permission key owns them (cluster
// credential metadata/rotation, Helm and install), so those remain admin-only.
//
// The released keyMatch2 matcher lets a `GET /clusters/:id` policy match a
// static sibling at the same depth. K8s read routes are either static siblings
// or sole `:param` routes at their depth, so this registry only binds the
// routes whose depth has no static/parameter sibling. Detail read routes that
// would widen another action stay unbound until Base's static-route guard is
// released and consumed.
func registerK8sPermissionRoutes() error {
	clusterDetail := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id"},
	}
	namespaceRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/namespaces"},
	}
	workloadRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/workloads"},
	}
	workloadPods := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/workloads/:namespace/:kind/:name/pods"},
	}
	configMapRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/configmaps"},
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/configmaps/:name"},
	}
	secretRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/secrets"},
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/secrets/:name"},
	}
	serviceRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/services"},
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
	}
	releaseRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/releases"},
	}
	ingressRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/ingresses"},
	}
	storageRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/storage/pvs"},
		{Method: "GET", Path: "/api/v1/business/k8s/storage/pvcs"},
		{Method: "GET", Path: "/api/v1/business/k8s/storage/storage-classes"},
	}
	backupRead := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/backup/velero/status"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/locations"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/locations/:id"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/backups"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/backups/:id"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/schedules"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/schedules/:id"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/restores"},
		{Method: "GET", Path: "/api/v1/business/k8s/backup/restores/:id"},
	}

	return permissionroutes.Register("business:k8s", []permissionroutes.Binding{
		{Permission: "business:k8s:cluster:list", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/k8s/clusters"},
		}},
		{Permission: "business:k8s:cluster:view", Routes: clusterDetail},
		{Permission: "business:k8s:cluster:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters"},
		}},
		{Permission: "business:k8s:cluster:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/k8s/clusters/:id"},
		}},
		{Permission: "business:k8s:cluster:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id"},
		}},
		{Permission: "business:k8s:cluster:sync", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/sync"},
		}},
		{Permission: "business:k8s:cluster:nodes", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/nodes"},
		}},
		{Permission: "business:k8s:namespace:view", Routes: namespaceRead},
		{Permission: "business:k8s:namespace:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/namespaces"},
		}},
		{Permission: "business:k8s:namespace:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id/namespaces/:name"},
		}},
		{Permission: "business:k8s:workload:list", Routes: workloadRead},
		{Permission: "business:k8s:workload:view", Routes: workloadPods},
		{Permission: "business:k8s:workload:scale", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/workloads/:namespace/:kind/:name/scale"},
		}},
		{Permission: "business:k8s:workload:restart", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/workloads/:namespace/:kind/:name/restart"},
		}},
		{Permission: "business:k8s:workload:logs", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/pods/:namespace/:podName/logs"},
		}},
		{Permission: "business:k8s:configmap:view", Routes: configMapRead},
		{Permission: "business:k8s:configmap:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/configmaps"},
		}},
		{Permission: "business:k8s:configmap:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id/configmaps/:name"},
		}},
		{Permission: "business:k8s:secret:view", Routes: secretRead},
		{Permission: "business:k8s:secret:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/secrets"},
		}},
		{Permission: "business:k8s:secret:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id/secrets/:name"},
		}},
		{Permission: "business:k8s:service:list", Routes: serviceRead},
		{Permission: "business:k8s:service:view", Routes: serviceRead},
		{Permission: "business:k8s:service:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/services"},
		}},
		{Permission: "business:k8s:service:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
		}},
		{Permission: "business:k8s:service:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
		}},
		{Permission: "business:k8s:release:list", Routes: releaseRead},
		{Permission: "business:k8s:release:view", Routes: releaseRead},
		{Permission: "business:k8s:release:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/releases"},
		}},
		{Permission: "business:k8s:release:rollback", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/releases/:id/rollback"},
		}},
		{Permission: "business:k8s:release:reconcile", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/releases/:id/reconcile"},
		}},
		{Permission: "business:k8s:ingress:list", Routes: ingressRead},
		{Permission: "business:k8s:ingress:view", Routes: ingressRead},
		{Permission: "business:k8s:ingress:update", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/ingresses/:id/sync"},
		}},
		{Permission: "business:k8s:ingress:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/ingresses/:id"},
		}},
		{Permission: "business:k8s:storage:list", Routes: storageRead},
		{Permission: "business:k8s:storage:view", Routes: storageRead},
		{Permission: "business:k8s:storage:update", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/storage/pvs/sync"},
			{Method: "POST", Path: "/api/v1/business/k8s/storage/pvcs/sync"},
			{Method: "POST", Path: "/api/v1/business/k8s/storage/storage-classes/sync"},
		}},
		{Permission: "business:k8s:storage:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/storage/pvs/:id"},
			{Method: "DELETE", Path: "/api/v1/business/k8s/storage/pvcs/:id"},
		}},
		{Permission: "business:k8s:backup:list", Routes: backupRead},
	})
}
