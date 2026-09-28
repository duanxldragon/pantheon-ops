package dynamicmodule

import (
	"encoding/json"
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"os"
	"path/filepath"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/internal/scaffold"
)

const msgModuleSchemaInvalid = "module.register.schema_invalid"

func generatedModuleRelativePath(parts ...string) (string, bool) {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.Trim(strings.ReplaceAll(strings.TrimSpace(part), "\\", "/"), "/")
		if normalized == "" || !filepath.IsLocal(filepath.FromSlash(normalized)) {
			return "", false
		}
		for _, segment := range strings.Split(normalized, "/") {
			if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, "..") || strings.ContainsAny(segment, `<>:"|?*`) {
				return "", false
			}
			cleaned = append(cleaned, segment)
		}
	}
	relativePath := filepath.ToSlash(filepath.Join(cleaned...))
	if relativePath == "" || !filepath.IsLocal(filepath.FromSlash(relativePath)) {
		return "", false
	}
	return relativePath, true
}

func (s *DynamicModuleService) RebuildGeneratedRegistries() error {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return common.NewNotFound("workspace.not_found")
	}
	if _, err := s.syncGeneratedModuleRegistrations(); err != nil {
		return err
	}
	_, _, err := s.refreshGeneratedWorkspaceArtifacts()
	return err
}

func (s *DynamicModuleService) listGeneratedModuleRefs() ([]scaffold.GeneratedModuleRef, error) {
	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> '' AND status <> ?", ModuleStatusUninstalled).Order("scope ASC").Order("name ASC").Find(&modules).Error; err != nil {
		return nil, err
	}
	refs := make([]scaffold.GeneratedModuleRef, 0, len(modules))
	for _, module := range modules {
		scope, name, err := splitModuleKey(module.Name)
		if err != nil {
			continue
		}
		if !s.generatedModuleArtifactsExist(scope, name) {
			continue
		}
		refs = append(refs, scaffold.GeneratedModuleRef{Name: name, Scope: scope})
	}
	return refs, nil
}

func (s *DynamicModuleService) generatedModuleArtifactsExist(scope, name string) bool {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return false
	}
	backendRelativePath, ok := generatedModuleRelativePath("backend", "modules", scope, name)
	if !ok {
		return false
	}
	frontendRelativePath, ok := generatedModuleRelativePath("frontend", "src", "modules", scope, name)
	if !ok {
		return false
	}
	schemaRelativePath, ok := generatedModuleRelativePath("schema", "generated", scope, name+".json")
	if !ok {
		return false
	}
	return generatedDirExists(s.workspaceRoot, backendRelativePath) &&
		generatedDirExists(s.workspaceRoot, frontendRelativePath) &&
		generatedPathExists(s.workspaceRoot, schemaRelativePath)
}

func (s *DynamicModuleService) loadGeneratedModuleSchema(scope, name string) (*scaffold.ModuleSchema, error) {
	relativeTarget, ok := generatedModuleRelativePath("schema", "generated", scope, name+".json")
	if !ok {
		return nil, common.NewBadRequest(msgModuleSchemaInvalid)
	}
	target, resolved := resolveGeneratedWorkspacePath(s.workspaceRoot, relativeTarget)
	if !resolved {
		return nil, common.NewBadRequest(msgModuleSchemaInvalid)
	}
	if !filepath.IsLocal(relativeTarget) {
		return nil, common.NewBadRequest(msgModuleSchemaInvalid)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, common.NewBadRequest("module.register.source_missing")
		}
		return nil, err
	}
	var schema scaffold.ModuleSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		return nil, common.NewBadRequest(msgModuleSchemaInvalid)
	}
	if strings.TrimSpace(schema.Name) == "" || strings.TrimSpace(schema.Scope) == "" || strings.TrimSpace(schema.Model.TableName) == "" {
		return nil, common.NewBadRequest(msgModuleSchemaInvalid)
	}
	return &schema, nil
}

func resolveGeneratedWorkspacePath(workspaceRoot, relativePath string) (string, bool) {
	normalizedRoot := filepath.Clean(strings.TrimSpace(workspaceRoot))
	normalizedRelative := filepath.ToSlash(strings.TrimSpace(relativePath))
	if normalizedRoot == "" || normalizedRelative == "" {
		return "", false
	}
	if strings.Contains(normalizedRelative, "..") || !filepath.IsLocal(normalizedRelative) {
		return "", false
	}
	target := filepath.Join(normalizedRoot, filepath.FromSlash(normalizedRelative))
	relativeToRoot, err := filepath.Rel(normalizedRoot, target)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return target, true
}

// openWorkspaceRoot opens the workspace directory through os.Root, the
// kernel-backed traversal-safe API (Go 1.24+): lookups are anchored at the
// root and cannot escape it, so request-controlled relative paths passed to
// Root.ReadFile cannot address files outside the workspace regardless of
// their content (".." segments, absolute paths, symlinks). This makes the
// containment guarantee structural instead of relying on path validation
// only, and keeps gosecurity:S2083 from reporting a traversal flow.
func openWorkspaceRoot(workspaceRoot string) (*os.Root, bool) {
	root := filepath.Clean(strings.TrimSpace(workspaceRoot))
	if root == "" {
		return nil, false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	workspace, err := os.OpenRoot(root)
	if err != nil {
		return nil, false
	}
	return workspace, true
}

func generatedPathExists(workspaceRoot, relativePath string) bool {
	path, ok := resolveGeneratedWorkspacePath(workspaceRoot, relativePath)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func generatedDirExists(workspaceRoot, relativePath string) bool {
	path, ok := resolveGeneratedWorkspacePath(workspaceRoot, relativePath)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func generatedFileContainsAll(workspaceRoot string, relativePath string, fragments ...string) bool {
	root, ok := openWorkspaceRoot(workspaceRoot)
	if !ok {
		return false
	}
	defer func() { _ = root.Close() }()
	normalizedRelative := filepath.ToSlash(strings.TrimSpace(relativePath))
	if normalizedRelative == "" || strings.Contains(normalizedRelative, "..") || !filepath.IsLocal(normalizedRelative) {
		return false
	}
	content, err := root.ReadFile(normalizedRelative)
	if err != nil {
		return false
	}
	body := string(content)
	for _, fragment := range fragments {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(body, fragment) {
			return false
		}
	}
	return true
}
