package filedist

import (
	"time"

	"gorm.io/gorm"
)

const (
	fileDistModuleKey      = "business.filedist"
	fileDistMenuTitleKey   = "operations.filedist.menu"
	fileDistMenuPath       = "/business/filedist/files"
	fileDistRouteName      = "filedist-file-list"
	fileDistComponentKey   = "business/filedist/FiledistList"
	fileDistPagePermission = "business:filedist:list"
)

type fileDistI18nSeed struct {
	Locale string
	Group  string
	Key    string
	Value  string
}

type fileDistMenuSeed struct {
	Key       string
	TitleKey  string
	Path      string
	Component string
	PagePerm  string
	Type      string
	Icon      string
	RouteName string
	Module    string
	Sort      int
}

var fileDistMenuSeeds = []fileDistMenuSeed{
	{Key: "filedist-files", TitleKey: fileDistMenuTitleKey, Path: fileDistMenuPath, Component: fileDistComponentKey, PagePerm: fileDistPagePermission, Type: "C", Icon: "storage", RouteName: fileDistRouteName, Module: fileDistModuleKey, Sort: 5},
}

var fileDistRuntimeI18nSeeds = []fileDistI18nSeed{
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.executionInterrupted", Value: "执行已中断；任务进入失败状态，请核对结果后重试"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.executionInterrupted", Value: "Execution was interrupted. Review the results before retrying."},
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.transferFailed", Value: "传输失败；请核对目标结果后重试"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.transferFailed", Value: "Transfer failed. Review target results before retrying."},
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.executionBusy", Value: "另一执行进程正在管理文件分发，请稍后重试"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.executionBusy", Value: "Another process owns file distribution. Try again later."},
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.notDue", Value: "任务尚未到计划执行时间"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.notDue", Value: "The task is not due yet."},
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.notCancellable", Value: "当前任务状态不允许取消"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.notCancellable", Value: "The task cannot be cancelled in its current state."},
	{Locale: "zh-CN", Group: "error", Key: "business.filedist.task.invalidContext", Value: "执行上下文无效，任务未开始"},
	{Locale: "en-US", Group: "error", Key: "business.filedist.task.invalidContext", Value: "The execution context is invalid. The task has not started."},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.createTask", Value: "创建分发任务"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.createSuccess", Value: "分发任务已创建"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.createFailed", Value: "创建失败，请检查文件、目标与权限后重试"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.createForbidden", Value: "当前权限不允许此操作"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.selectionRequired", Value: "请重新解析当前业务范围内的可用目标"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.required", Value: "请填写任务名称、目标路径及 SSH 认证信息"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.invalidSchedule", Value: "请选择有效的计划执行时间"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.selectFile", Value: "分发文件"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.selectFilePlaceholder", Value: "选择已上传的文件"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.noFileFirst", Value: "请先在文件库上传文件"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.targets", Value: "授权目标"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.targetConfig", Value: "传输配置"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.sshUsername", Value: "SSH 用户名"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.usernamePlaceholder", Value: "例如 ops"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.pathPlaceholder", Value: "例如 /opt/app"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.authMode", Value: "SSH 认证方式"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.password", Value: "SSH 密码"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.privateKey", Value: "SSH 私钥"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.hostFingerprintFor", Value: "{{host}} 的 SSH 主机指纹"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.invalidFingerprint", Value: "请输入有效的 SHA256 SSH 主机指纹"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.fingerprintPlaceholder", Value: "SHA256 主机指纹"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.maxRetries", Value: "最大重试次数"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.scheduledAt", Value: "计划执行时间"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.endpointHint", Value: "主机地址和端口由授权目标解析确定。请独立核实每台主机指纹；认证信息用于本次任务，详情不会回显。"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.create.optionsFailed", Value: "无法加载文件或目标选项，请重试"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.backToTasks", Value: "返回任务列表"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.start", Value: "开始分发"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.cancelConfirm", Value: "停止此任务的后续传输？已写入目标的文件内容不会自动撤回。"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.refreshFailed", Value: "刷新失败，当前显示的是上次成功读取的结果"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.noTargets", Value: "暂无目标记录"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.noEvents", Value: "暂无事件记录"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.pending", Value: "待执行"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.running", Value: "执行中"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.transferring", Value: "传输中"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.retrying", Value: "重试中"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.completed", Value: "已完成"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.partial", Value: "部分成功"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.failed", Value: "失败"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.cancelled", Value: "已取消"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.active", Value: "可用"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.status.unknown", Value: "未知"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.storage.local", Value: "本地存储"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.storage.s3", Value: "S3 存储"},
	{Locale: "zh-CN", Group: "business", Key: "business.filedist.storage.minio", Value: "MinIO 存储"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.createTask", Value: "Create distribution task"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.createSuccess", Value: "Distribution task created"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.createFailed", Value: "Creation failed. Check the file, targets and permissions before retrying."},
	{Locale: "en-US", Group: "business", Key: "business.filedist.createForbidden", Value: "Your current permissions do not allow this action"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.selectionRequired", Value: "Resolve eligible targets for the current business scope again"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.required", Value: "Enter a task name, target path and SSH authentication"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.invalidSchedule", Value: "Choose a valid scheduled time"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.selectFile", Value: "Distribution file"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.selectFilePlaceholder", Value: "Choose an uploaded file"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.noFileFirst", Value: "Upload a file to the library first"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.targets", Value: "Authorized targets"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.targetConfig", Value: "Transfer configuration"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.sshUsername", Value: "SSH username"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.usernamePlaceholder", Value: "For example, ops"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.pathPlaceholder", Value: "For example, /opt/app"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.authMode", Value: "SSH authentication"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.password", Value: "SSH password"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.privateKey", Value: "SSH private key"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.hostFingerprintFor", Value: "SSH host fingerprint for {{host}}"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.invalidFingerprint", Value: "Enter a valid SHA256 SSH host fingerprint"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.fingerprintPlaceholder", Value: "SHA256 host fingerprint"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.maxRetries", Value: "Maximum retries"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.scheduledAt", Value: "Scheduled time"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.endpointHint", Value: "Authorized target resolution determines host addresses and ports. Verify each host fingerprint independently. Authentication is used for this task and omitted from detail responses."},
	{Locale: "en-US", Group: "business", Key: "business.filedist.create.optionsFailed", Value: "Could not load file or target options. Retry the lookup."},
	{Locale: "en-US", Group: "business", Key: "business.filedist.backToTasks", Value: "Back to tasks"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.start", Value: "Start distribution"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.cancelConfirm", Value: "Stop further transfers for this task? Content already written to targets will not be automatically removed."},
	{Locale: "en-US", Group: "business", Key: "business.filedist.refreshFailed", Value: "Refresh failed. The last successfully loaded results are shown."},
	{Locale: "en-US", Group: "business", Key: "business.filedist.noTargets", Value: "No target records"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.noEvents", Value: "No event records"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.pending", Value: "Pending"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.running", Value: "Running"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.transferring", Value: "Transferring"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.retrying", Value: "Retrying"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.completed", Value: "Completed"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.partial", Value: "Partially successful"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.failed", Value: "Failed"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.cancelled", Value: "Cancelled"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.active", Value: "Available"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.status.unknown", Value: "Unknown"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.storage.local", Value: "Local storage"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.storage.s3", Value: "S3 storage"},
	{Locale: "en-US", Group: "business", Key: "business.filedist.storage.minio", Value: "MinIO storage"},
}

func seedFileDistMenusAndI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range fileDistMenuSeeds {
			var menuID uint64
			if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
			payload := map[string]any{
				"parent_id":  0,
				"title_key":  seed.TitleKey,
				"path":       seed.Path,
				"component":  seed.Component,
				"page_perm":  seed.PagePerm,
				"perms":      "",
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
					return err
				}
				if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
				return err
			}
			if err := ensureFileDistAdminBinding(tx, menuID, seed.PagePerm); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range append([]fileDistI18nSeed{
			{Locale: "zh-CN", Group: "menu", Key: fileDistMenuTitleKey, Value: "文件分发"},
			{Locale: "en-US", Group: "menu", Key: fileDistMenuTitleKey, Value: "File Distribution"},
		}, fileDistRuntimeI18nSeeds...) {
			payload := map[string]any{
				"module":           fileDistModuleKey,
				"locale":           item.Locale,
				"group_name":       item.Group,
				"key":              item.Key,
				"value":            item.Value,
				"lifecycle_status": "active",
				"updated_at":       time.Now(),
			}
			var count int64
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", fileDistModuleKey, item.Locale, item.Key).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", fileDistModuleKey, item.Locale, item.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureFileDistAdminBinding(db *gorm.DB, menuID uint64, pagePermission string) error {
	if menuID == 0 || !db.Migrator().HasTable("system_role") {
		return nil
	}
	var roleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		return nil
	}
	if db.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if pagePermission == "" || !db.Migrator().HasTable("system_role_permission") {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", roleID, pagePermission).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", roleID, pagePermission).Error
	}
	return nil
}
