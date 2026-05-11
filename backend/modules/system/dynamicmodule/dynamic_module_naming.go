package dynamicmodule

import (
	"errors"
	"strings"
	"unicode"
)

func inferModuleScope(moduleName string) string {
	if strings.HasPrefix(moduleName, "business.") {
		return "business"
	}
	if strings.HasPrefix(moduleName, "platform") {
		return "platform"
	}
	return "system"
}

func splitModuleKey(moduleName string) (string, string, error) {
	normalized := strings.TrimSpace(moduleName)
	if normalized == "" {
		return "", "", errors.New("module.invalid_name")
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("module.invalid_name")
	}
	scope := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(strings.ReplaceAll(parts[1], ".", "/"))
	if (scope != "system" && scope != "business") || name == "" {
		return "", "", errors.New("module.invalid_name")
	}
	return scope, name, nil
}

func buildModuleKey(scope string, name string) string {
	return strings.TrimSpace(scope) + "." + strings.ReplaceAll(strings.Trim(strings.TrimSpace(name), "/"), "/", ".")
}

func resolveGeneratedParentMenu(scope string, name string, explicitParent string) (string, string) {
	normalizedExplicit := normalizeGeneratedMenuPath(explicitParent)
	if normalizedExplicit != "" {
		return normalizedExplicit, "explicit"
	}
	if scope == "business" {
		segments := strings.Split(strings.Trim(name, "/"), "/")
		if len(segments) > 1 {
			return "/business/" + segments[0], "inferred"
		}
	}
	return "", "top_level"
}

func inferGeneratedModelName(name string, explicit string) string {
	trimmed := strings.TrimSpace(explicit)
	if trimmed != "" {
		return trimmed
	}
	return toGeneratedPascal(name)
}

func normalizeGeneratedModulePath(name string) string {
	return strings.Trim(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"), "/")
}

func normalizeGeneratedMenuPath(path string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if normalized == "" {
		return ""
	}
	return "/" + strings.Trim(normalized, "/")
}

func toGeneratedPascal(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == '/' || r == '.'
	})
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}
	return builder.String()
}

func inferStaticModuleSource(moduleName string) string {
	if strings.HasPrefix(moduleName, "platform") || moduleName == "system" || strings.HasPrefix(moduleName, "system.") {
		return "core"
	}
	return "static"
}

func inferRegistrationSource(scope string, sourceMode string, name string, managed bool) string {
	normalizedSourceMode := strings.TrimSpace(sourceMode)
	if normalizedSourceMode == "database" || normalizedSourceMode == "manual" {
		return normalizedSourceMode
	}
	if managed {
		return "generated"
	}
	return inferStaticModuleSource(buildModuleKey(scope, name))
}
