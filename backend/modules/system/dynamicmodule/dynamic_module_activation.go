package dynamicmodule

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *DynamicModuleService) AuditPendingGeneratedModuleActivations() (*ActivationAuditSummary, error) {
	if s.db == nil {
		return &ActivationAuditSummary{}, nil
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, fmt.Errorf("workspace.not_found")
	}

	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> '' AND status = ?", ModuleStatusPendingActivation).Find(&modules).Error; err != nil {
		return nil, err
	}

	summary := &ActivationAuditSummary{}
	for index := range modules {
		module := &modules[index]
		scope, name, err := splitModuleKey(module.Name)
		if err != nil {
			continue
		}

		summary.CheckedModules++
		backendReady := s.generatedModuleRuntimeReady(module.Name)
		frontendReady := s.generatedFrontendBundleReady(scope, name)
		if backendReady {
			summary.RuntimeReadyModules++
		}
		if frontendReady {
			summary.FrontendReadyModules++
		}

		status := ModuleStatusPendingActivation
		if backendReady && frontendReady {
			status = ModuleStatusActive
			summary.ActivatedModules++
		} else {
			summary.PendingModules++
		}

		if err := s.persistModuleDiagnostics(module, status, "", s.buildActivationAuditVerifications(module.Name, scope, name, backendReady, frontendReady)); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

func (s *DynamicModuleService) buildActivationAuditVerifications(moduleKey string, scope string, name string, backendReady bool, frontendReady bool) []GeneratedModuleVerification {
	verifications := make([]GeneratedModuleVerification, 0, 6)
	if s.generatedModuleArtifactsExist(scope, name) {
		verifications = append(verifications, GeneratedModuleVerification{
			Code:       "artifact_check",
			Status:     "pass",
			MessageKey: "module.generate.verify.source_written",
			Detail:     filepath.ToSlash(filepath.Join("modules", scope, name)),
		})
	} else {
		return []GeneratedModuleVerification{buildArtifactMissingVerification(scope, name)}
	}

	if backendReady {
		verifications = append(verifications, GeneratedModuleVerification{
			Code:       "backend_runtime_ready",
			Status:     "pass",
			MessageKey: "module.generate.verify.backend_runtime_ready",
			Detail:     moduleKey,
		})
	} else {
		verifications = append(verifications,
			GeneratedModuleVerification{
				Code:       "backend_runtime_ready",
				Status:     "info",
				MessageKey: "module.generate.verify.backend_runtime_pending",
				Detail:     moduleKey,
			},
			GeneratedModuleVerification{
				Code:       "restart_required",
				Status:     "info",
				MessageKey: "module.generate.verify.restart_required",
				Detail:     "backend_restart_required=true",
			},
		)
	}

	if frontendReady {
		verifications = append(verifications, GeneratedModuleVerification{
			Code:       "frontend_bundle_ready",
			Status:     "pass",
			MessageKey: "module.generate.verify.frontend_bundle_ready",
			Detail:     filepath.ToSlash(filepath.Join("frontend", "dist")),
		})
	} else {
		verifications = append(verifications,
			GeneratedModuleVerification{
				Code:       "frontend_bundle_ready",
				Status:     "info",
				MessageKey: "module.generate.verify.frontend_bundle_pending",
				Detail:     filepath.ToSlash(filepath.Join("frontend", "dist")),
			},
			GeneratedModuleVerification{
				Code:       "frontend_build_required",
				Status:     "info",
				MessageKey: "module.generate.verify.frontend_build_required",
				Detail:     "frontend_build_required=true",
			},
		)
	}

	if backendReady && frontendReady {
		verifications = append(verifications, GeneratedModuleVerification{
			Code:       "activation_ready",
			Status:     "pass",
			MessageKey: "module.generate.verify.activation_ready",
			Detail:     "status=active",
		})
	} else {
		verifications = append(verifications, GeneratedModuleVerification{
			Code:       "pending_activation",
			Status:     "info",
			MessageKey: "module.generate.verify.pending_activation",
			Detail:     "status=pending_activation",
		})
	}

	return verifications
}

func (s *DynamicModuleService) generatedModuleRuntimeReady(moduleKey string) bool {
	if s.db == nil || !s.db.Migrator().HasTable("system_menu") {
		return false
	}
	var count int64
	_ = s.db.Table("system_menu").Where("module = ?", moduleKey).Count(&count).Error
	return count > 0
}

func (s *DynamicModuleService) generatedFrontendBundleReady(scope string, name string) bool {
	moduleDir := filepath.Join(s.workspaceRoot, "frontend", "src", "modules", scope, name)
	distDir := filepath.Join(s.workspaceRoot, "frontend", "dist")
	moduleMtime, ok := newestFileModTime(moduleDir)
	if !ok {
		return false
	}
	distMtime, ok := newestFileModTime(distDir)
	if !ok {
		return false
	}
	return !distMtime.Before(moduleMtime)
}

func newestFileModTime(root string) (time.Time, bool) {
	info, err := os.Stat(root)
	if err != nil {
		return time.Time{}, false
	}
	if !info.IsDir() {
		return info.ModTime(), true
	}

	found := false
	latest := info.ModTime()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		fileInfo, err := d.Info()
		if err != nil {
			return nil
		}
		if !found || fileInfo.ModTime().After(latest) {
			latest = fileInfo.ModTime()
			found = true
		}
		return nil
	})
	return latest, found
}
