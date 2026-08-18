package service

import (
	"context"
	"errors"
	"strings"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/gorm"
)

const (
	errNotInitialized       = "business.service.database_not_initialized"
	errApplicationNotFound  = "business.service.application_not_found"
	errApplicationCodeExist = "business.service.application_code_exists"
	errApplicationInUse     = "business.service.application_in_use"
	errServiceNotFound      = "business.service.service_not_found"
	errServiceCodeExist     = "business.service.service_code_exists"
	errServiceInUse         = "business.service.service_in_use"
	errInstanceNotFound     = "business.service.instance_not_found"
	errTargetInvalid        = "business.service.target_invalid"
	errTargetNotFound       = "business.service.target_not_found"
	errTargetConflict       = "business.service.target_conflict"
	errScopeInvalid         = "business.service.scope_invalid"
)

// Manager owns application, service, and service-instance persistence operations.
type Manager struct {
	db         *gorm.DB
	bizScope   bizcap.BizScopeReader
	cmdbReader bizcap.CMDBHostReader
	k8sReader  bizcap.K8sTargetReader
}

// NewManager creates the service domain manager with its cross-module readers.
func NewManager(db *gorm.DB, deps Dependencies) *Manager {
	return &Manager{
		db:         db,
		bizScope:   deps.BizScopeReader,
		cmdbReader: deps.CMDBReader,
		k8sReader:  deps.K8sReader,
	}
}

// Migrate creates or upgrades the service domain schema.
func (m *Manager) Migrate() error {
	if m.db == nil {
		return errors.New(errNotInitialized)
	}
	return m.db.AutoMigrate(&Application{}, &Service{}, &ServiceInstance{})
}

// ListApplications returns applications visible in the supplied data scope.
func (m *Manager) ListApplications(q ApplicationQuery, scope *common.DataScopeReq) (*ApplicationListResponse, error) {
	if m.db == nil {
		return nil, errors.New(errNotInitialized)
	}
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	db := m.db.Model(&Application{}).Scopes(database.WithDataScope(scope))
	if key := strings.TrimSpace(q.Keyword); key != "" {
		like := "%" + key + "%"
		db = db.Where("code LIKE ? OR name LIKE ?", like, like)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []Application
	if err := db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ApplicationResponse, 0, len(rows))
	for i := range rows {
		item, err := m.applicationResponse(&rows[i])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &ApplicationListResponse{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// ListApplicationOptions returns active application options visible in the supplied data scope.
func (m *Manager) ListApplicationOptions(scope *common.DataScopeReq) ([]OptionItem, error) {
	var rows []Application
	if err := m.db.Model(&Application{}).Scopes(database.WithDataScope(scope)).
		Where("status = ?", StatusActive).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OptionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, OptionItem{ID: row.ID, Code: row.Code, Name: row.Name, Label: row.Name, Value: row.ID})
	}
	return items, nil
}

// CreateApplication creates an application under an active business scope.
func (m *Manager) CreateApplication(req CreateApplicationRequest, actor string, scope *common.DataScopeReq) (*ApplicationResponse, error) {
	if m.db == nil {
		return nil, errors.New(errNotInitialized)
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" || req.BusinessScopeID == 0 {
		return nil, errors.New(errTargetInvalid)
	}
	if m.bizScope == nil {
		return nil, errors.New("business.service.bizscope_reader_not_configured")
	}
	biz, err := m.bizScope.GetActive(context.Background(), req.BusinessScopeID, scope)
	if err != nil {
		return nil, errors.New(errScopeInvalid)
	}
	var count int64
	if err := m.db.Model(&Application{}).Where("code = ?", req.Code).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New(errApplicationCodeExist)
	}
	status := req.Status
	if status == "" {
		status = StatusActive
	}
	row := Application{
		Code: req.Code, Name: req.Name, BusinessScopeID: biz.ID, BusinessScopeName: biz.Name,
		DeptID: bizScopeDeptID(biz, scope), Status: status, Owner: strings.TrimSpace(req.Owner),
		Remark: req.Remark, CreatedBy: actor, UpdatedBy: actor,
	}
	if err := m.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp, err := m.applicationResponse(&row)
	return &resp, err
}

// GetApplication returns an application reference visible in the supplied data scope.
func (m *Manager) GetApplication(ctx context.Context, id uint64, scope *common.DataScopeReq) (ApplicationRef, error) {
	var row Application
	if err := m.db.WithContext(ctx).Model(&Application{}).Scopes(database.WithDataScope(scope)).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApplicationRef{}, errors.New(errApplicationNotFound)
		}
		return ApplicationRef{}, err
	}
	return ApplicationRef{ID: row.ID, Code: row.Code, Name: row.Name, BusinessScopeID: row.BusinessScopeID, DeptID: row.DeptID, Status: row.Status}, nil
}

// UpdateApplication updates an application visible in the supplied data scope.
func (m *Manager) UpdateApplication(id uint64, req UpdateApplicationRequest, actor string, scope *common.DataScopeReq) (*ApplicationResponse, error) {
	row, err := m.findApplication(id, scope)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_by": actor, "updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}
	if req.Owner != nil {
		updates["owner"] = strings.TrimSpace(*req.Owner)
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if err := m.db.Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	updated, err := m.findApplication(id, scope)
	if err != nil {
		return nil, err
	}
	resp, err := m.applicationResponse(updated)
	return &resp, err
}

// DeleteApplication deletes an application when it is not referenced by a service.
func (m *Manager) DeleteApplication(id uint64, scope *common.DataScopeReq) error {
	row, err := m.findApplication(id, scope)
	if err != nil {
		return err
	}
	var count int64
	if err := m.db.Model(&Service{}).Where("application_id = ?", row.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(errApplicationInUse)
	}
	if result := m.db.Delete(row); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return errors.New(errApplicationNotFound)
	}
	return nil
}

// ListServices returns services visible in the supplied data scope.
func (m *Manager) ListServices(q ServiceQuery, scope *common.DataScopeReq) (*ServiceListResponse, error) {
	if m.db == nil {
		return nil, errors.New(errNotInitialized)
	}
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	db := m.serviceQuery(scope)
	if q.ApplicationID > 0 {
		db = db.Where("application_id = ?", q.ApplicationID)
	}
	if q.Keyword != "" {
		like := "%" + strings.TrimSpace(q.Keyword) + "%"
		db = db.Where("code LIKE ? OR name LIKE ?", like, like)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []Service
	if err := db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ServiceResponse, 0, len(rows))
	for i := range rows {
		item, err := m.serviceResponse(&rows[i], scope)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &ServiceListResponse{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// ListServiceOptions returns active service options for an application.
func (m *Manager) ListServiceOptions(applicationID uint64, scope *common.DataScopeReq) ([]OptionItem, error) {
	db := m.serviceQuery(scope).Where("status = ?", StatusActive)
	if applicationID > 0 {
		db = db.Where("application_id = ?", applicationID)
	}
	var rows []Service
	if err := db.Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OptionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, OptionItem{ID: row.ID, Code: row.Code, Name: row.Name, Label: row.Name, Value: row.ID})
	}
	return items, nil
}

// CreateService creates a service for an application.
func (m *Manager) CreateService(req CreateServiceRequest, actor string, scope *common.DataScopeReq) (*ServiceResponse, error) {
	req.Code, req.Name, req.RuntimeType = strings.TrimSpace(req.Code), strings.TrimSpace(req.Name), strings.TrimSpace(req.RuntimeType)
	if req.ApplicationID == 0 || req.Code == "" || req.Name == "" || req.RuntimeType == "" {
		return nil, errors.New(errTargetInvalid)
	}
	app, err := m.findApplication(req.ApplicationID, scope)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := m.db.Model(&Service{}).Where("application_id = ? AND code = ?", app.ID, req.Code).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New(errServiceCodeExist)
	}
	status := req.Status
	if status == "" {
		status = StatusActive
	}
	row := Service{ApplicationID: app.ID, Code: req.Code, Name: req.Name, RuntimeType: req.RuntimeType, Description: req.Description, Status: status, CreatedBy: actor, UpdatedBy: actor}
	if err := m.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp, err := m.serviceResponse(&row, scope)
	return &resp, err
}

// GetService returns a service reference visible in the supplied data scope.
func (m *Manager) GetService(ctx context.Context, id uint64, scope *common.DataScopeReq) (ServiceRef, error) {
	var row Service
	if err := m.serviceQuery(scope).WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ServiceRef{}, errors.New(errServiceNotFound)
		}
		return ServiceRef{}, err
	}
	app, err := m.findApplication(row.ApplicationID, scope)
	if err != nil {
		return ServiceRef{}, err
	}
	return ServiceRef{ID: row.ID, ApplicationID: row.ApplicationID, Code: row.Code, Name: row.Name, RuntimeType: row.RuntimeType, BusinessScopeID: app.BusinessScopeID, DeptID: app.DeptID, Status: row.Status}, nil
}

// UpdateService updates a service visible in the supplied data scope.
func (m *Manager) UpdateService(id uint64, req UpdateServiceRequest, actor string, scope *common.DataScopeReq) (*ServiceResponse, error) {
	row, err := m.findService(id, scope)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_by": actor, "updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.RuntimeType != nil {
		updates["runtime_type"] = strings.TrimSpace(*req.RuntimeType)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}
	if err := m.db.Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	resp, err := m.serviceResponse(row, scope)
	return &resp, err
}

// DeleteService deletes a service when it has no service instances.
func (m *Manager) DeleteService(id uint64, scope *common.DataScopeReq) error {
	row, err := m.findService(id, scope)
	if err != nil {
		return err
	}
	var count int64
	if err := m.db.Model(&ServiceInstance{}).Where("service_id = ?", row.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(errServiceInUse)
	}
	return m.db.Delete(row).Error
}

// ListInstances returns service instances visible in the supplied data scope.
func (m *Manager) ListInstances(q InstanceQuery, scope *common.DataScopeReq) (*InstanceListResponse, error) {
	if m.db == nil {
		return nil, errors.New(errNotInitialized)
	}
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	db := m.instanceQuery(scope)
	if q.ServiceID > 0 {
		db = db.Where("service_id = ?", q.ServiceID)
	}
	if q.TargetType != "" {
		db = db.Where("target_type = ?", q.TargetType)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []ServiceInstance
	if err := db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]InstanceResponse, 0, len(rows))
	for i := range rows {
		item, err := m.instanceResponse(&rows[i], scope)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &InstanceListResponse{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// CreateInstance creates a VM or Kubernetes-targeted service instance.
func (m *Manager) CreateInstance(ctx context.Context, req CreateInstanceRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error) {
	serviceRef, err := m.GetService(ctx, req.ServiceID, scope)
	if err != nil {
		return InstanceRef{}, err
	}
	if serviceRef.Status != StatusActive {
		return InstanceRef{}, errors.New("business.service.service_inactive")
	}
	req.Environment = strings.TrimSpace(req.Environment)
	if req.Environment == "" {
		return InstanceRef{}, errors.New(errTargetInvalid)
	}
	row := ServiceInstance{
		ServiceID: req.ServiceID, Environment: req.Environment, TargetType: strings.TrimSpace(req.TargetType),
		HostID: req.HostID, K8sClusterID: req.K8sClusterID, Namespace: strings.TrimSpace(req.Namespace),
		WorkloadKind: strings.TrimSpace(req.WorkloadKind), WorkloadName: strings.TrimSpace(req.WorkloadName),
		DesiredVersion: req.DesiredVersion, Status: req.Status, CreatedBy: actor, UpdatedBy: actor,
	}
	if row.Status == "" {
		row.Status = InstanceStatusActive
	}
	row.DesiredState = DesiredStateStopped
	row.ObservedState = ObservedStateUnknown
	row.HealthState = HealthStateUnknown
	if err := m.validateAndNormalizeTarget(ctx, &row, serviceRef.BusinessScopeID, scope); err != nil {
		return InstanceRef{}, err
	}
	if err := m.ensureTargetUnique(&row); err != nil {
		return InstanceRef{}, err
	}
	if err := m.db.Create(&row).Error; err != nil {
		return InstanceRef{}, err
	}
	return m.instanceRef(&row, serviceRef), nil
}

// UpdateInstance updates a service instance visible in the supplied data scope.
func (m *Manager) UpdateInstance(id uint64, req UpdateInstanceRequest, actor string, scope *common.DataScopeReq) (*InstanceResponse, error) {
	row, err := m.findInstance(id, scope)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_by": actor, "updated_at": time.Now(), "lifecycle_version": gorm.Expr("lifecycle_version + 1")}
	if req.DesiredVersion != nil {
		updates["desired_version"] = *req.DesiredVersion
	}
	if req.CurrentVersion != nil {
		updates["current_version"] = *req.CurrentVersion
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if err := m.db.Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	resp, err := m.instanceResponse(row, scope)
	return &resp, err
}

// GetInstance returns a service instance reference visible in the supplied data scope.
func (m *Manager) GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (InstanceRef, error) {
	row, err := m.findInstance(id, scope)
	if err != nil {
		return InstanceRef{}, err
	}
	svc, err := m.GetService(ctx, row.ServiceID, scope)
	if err != nil {
		return InstanceRef{}, err
	}
	return m.instanceRef(row, svc), nil
}

// DeleteInstance deletes a service instance visible in the supplied data scope.
func (m *Manager) DeleteInstance(id uint64, scope *common.DataScopeReq) error {
	row, err := m.findInstance(id, scope)
	if err != nil {
		return err
	}
	return m.db.Delete(row).Error
}

func (m *Manager) validateAndNormalizeTarget(ctx context.Context, row *ServiceInstance, scopeID uint64, dataScope *common.DataScopeReq) error {
	switch strings.ToLower(row.TargetType) {
	case TargetTypeVM:
		if row.HostID == 0 || row.K8sClusterID != 0 || row.Namespace != "" || row.WorkloadKind != "" || row.WorkloadName != "" {
			return errors.New(errTargetInvalid)
		}
		if m.cmdbReader == nil {
			return errors.New("business.service.cmdb_reader_not_configured")
		}
		hosts, err := m.cmdbReader.GetByIDs(ctx, bizcap.HostIDsQuery{HostIDs: []uint64{row.HostID}, DataScope: dataScope})
		if err != nil {
			return err
		}
		if len(hosts.Items) != 1 || hosts.Items[0].BusinessScopeID != scopeID {
			return errors.New(errTargetNotFound)
		}
	case TargetTypeK8s:
		if row.K8sClusterID == 0 || row.Namespace == "" || row.WorkloadKind == "" || row.WorkloadName == "" || row.HostID != 0 {
			return errors.New(errTargetInvalid)
		}
		if m.k8sReader == nil {
			return errors.New("business.service.k8s_reader_not_configured")
		}
		target, err := m.k8sReader.ResolveTarget(ctx, bizcap.K8sTargetQuery{
			ClusterID: row.K8sClusterID, Namespace: row.Namespace, WorkloadKind: row.WorkloadKind,
			WorkloadName: row.WorkloadName, DataScope: dataScope,
		})
		if err != nil {
			return err
		}
		if target.BusinessScopeID != scopeID {
			return errors.New(errTargetNotFound)
		}
		row.WorkloadKind = target.WorkloadKind
	default:
		return errors.New(errTargetInvalid)
	}
	return nil
}

func (m *Manager) ensureTargetUnique(row *ServiceInstance) error {
	db := m.db.Model(&ServiceInstance{}).Where("service_id = ?", row.ServiceID)
	switch row.TargetType {
	case TargetTypeVM:
		db = db.Where("target_type = ? AND host_id = ?", TargetTypeVM, row.HostID)
	case TargetTypeK8s:
		db = db.Where("target_type = ? AND k8s_cluster_id = ? AND namespace = ? AND workload_kind = ? AND workload_name = ?",
			TargetTypeK8s, row.K8sClusterID, row.Namespace, row.WorkloadKind, row.WorkloadName)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(errTargetConflict)
	}
	return nil
}

func (m *Manager) serviceQuery(scope *common.DataScopeReq) *gorm.DB {
	db := m.db.Model(&Service{}).Where("application_id IN (?)",
		m.db.Model(&Application{}).Scopes(database.WithDataScope(scope)).Select("id"))
	return db
}

func (m *Manager) instanceQuery(scope *common.DataScopeReq) *gorm.DB {
	return m.db.Model(&ServiceInstance{}).Where("service_id IN (?)",
		m.serviceQuery(scope).Select("id"))
}

func (m *Manager) findApplication(id uint64, scope *common.DataScopeReq) (*Application, error) {
	var row Application
	if err := m.db.Model(&Application{}).Scopes(database.WithDataScope(scope)).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errApplicationNotFound)
		}
		return nil, err
	}
	return &row, nil
}

func (m *Manager) findService(id uint64, scope *common.DataScopeReq) (*Service, error) {
	var row Service
	if err := m.serviceQuery(scope).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errServiceNotFound)
		}
		return nil, err
	}
	return &row, nil
}

func (m *Manager) findInstance(id uint64, scope *common.DataScopeReq) (*ServiceInstance, error) {
	var row ServiceInstance
	if err := m.instanceQuery(scope).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errInstanceNotFound)
		}
		return nil, err
	}
	return &row, nil
}

func (m *Manager) applicationResponse(row *Application) (ApplicationResponse, error) {
	var count int64
	if err := m.db.Model(&Service{}).Where("application_id = ?", row.ID).Count(&count).Error; err != nil {
		return ApplicationResponse{}, err
	}
	return ApplicationResponse{ID: row.ID, Code: row.Code, Name: row.Name, BusinessScopeID: row.BusinessScopeID, BusinessScopeName: row.BusinessScopeName, DeptID: row.DeptID, Status: row.Status, Owner: row.Owner, Remark: row.Remark, ServiceCount: count, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (m *Manager) serviceResponse(row *Service, scope *common.DataScopeReq) (ServiceResponse, error) {
	app, err := m.findApplication(row.ApplicationID, scope)
	if err != nil {
		return ServiceResponse{}, err
	}
	var count int64
	if err := m.db.Model(&ServiceInstance{}).Where("service_id = ?", row.ID).Count(&count).Error; err != nil {
		return ServiceResponse{}, err
	}
	return ServiceResponse{ID: row.ID, ApplicationID: row.ApplicationID, ApplicationCode: app.Code, ApplicationName: app.Name, BusinessScopeID: app.BusinessScopeID, Code: row.Code, Name: row.Name, RuntimeType: row.RuntimeType, Description: row.Description, Status: row.Status, InstanceCount: count, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (m *Manager) instanceResponse(row *ServiceInstance, scope *common.DataScopeReq) (InstanceResponse, error) {
	normalizeInstanceState(row)
	svc, err := m.GetService(context.Background(), row.ServiceID, scope)
	if err != nil {
		return InstanceResponse{}, err
	}
	return InstanceResponse{ID: row.ID, ServiceID: row.ServiceID, ServiceCode: svc.Code, ServiceName: svc.Name, ApplicationID: svc.ApplicationID, BusinessScopeID: svc.BusinessScopeID, DeptID: svc.DeptID, Environment: row.Environment, TargetType: row.TargetType, HostID: row.HostID, K8sClusterID: row.K8sClusterID, Namespace: row.Namespace, WorkloadKind: row.WorkloadKind, WorkloadName: row.WorkloadName, DesiredVersion: row.DesiredVersion, CurrentVersion: row.CurrentVersion, RollbackVersion: row.RollbackVersion, DesiredState: row.DesiredState, ObservedState: row.ObservedState, HealthState: row.HealthState, HealthObservedAt: row.HealthObservedAt, HealthMessage: row.HealthMessage, HealthRevision: row.HealthRevision, LastTransitionID: row.LastTransitionID, LastTransitionAt: row.LastTransitionAt, LifecycleVersion: row.LifecycleVersion, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (m *Manager) instanceRef(row *ServiceInstance, svc ServiceRef) InstanceRef {
	normalizeInstanceState(row)
	return InstanceRef{ID: row.ID, ServiceID: row.ServiceID, ApplicationID: svc.ApplicationID, BusinessScopeID: svc.BusinessScopeID, DeptID: svc.DeptID, Environment: row.Environment, TargetType: row.TargetType, HostID: row.HostID, K8sClusterID: row.K8sClusterID, Namespace: row.Namespace, WorkloadKind: row.WorkloadKind, WorkloadName: row.WorkloadName, DesiredVersion: row.DesiredVersion, CurrentVersion: row.CurrentVersion, RollbackVersion: row.RollbackVersion, DesiredState: row.DesiredState, ObservedState: row.ObservedState, HealthState: row.HealthState, HealthMessage: row.HealthMessage, HealthRevision: row.HealthRevision, LastTransitionID: row.LastTransitionID, LifecycleVersion: row.LifecycleVersion, Status: row.Status}
}

func normalizeInstanceState(row *ServiceInstance) {
	if row.DesiredState == "" {
		row.DesiredState = DesiredStateStopped
	}
	if row.ObservedState == "" {
		row.ObservedState = ObservedStateUnknown
	}
	if row.HealthState == "" {
		row.HealthState = HealthStateUnknown
	}
}

func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	return page, size
}

func bizScopeDeptID(ref bizcap.BizScopeRef, scope *common.DataScopeReq) uint64 {
	if ref.DeptID > 0 {
		return ref.DeptID
	}
	if scope != nil && scope.DeptID > 0 && !scope.IsAdmin && scope.Mode == common.DataScopeModeDept {
		return scope.DeptID
	}
	// BizScopeRef intentionally stays a small cross-module contract. When the
	// owner does not expose dept_id, the request scope is the only safe fallback.
	if scope != nil && scope.DeptID > 0 {
		return scope.DeptID
	}
	return 0
}

var _ Reader = (*Manager)(nil)
var _ Command = (*Manager)(nil)
