package dynamicmodule

import (
	"errors"
	"strings"
	"unicode"
)

const errInvalidModuleName = "module.invalid_name"

func normalizeStaticModuleName(moduleName string) string {
	normalized := strings.TrimSpace(moduleName)
	switch normalized {
	case "platform.lowcode":
		return "system.lowcode"
	default:
		return normalized
	}
}

func inferModuleScope(moduleName string) string {
	moduleName = normalizeStaticModuleName(moduleName)
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
		return "", "", errors.New(errInvalidModuleName)
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New(errInvalidModuleName)
	}
	scope := strings.TrimSpace(parts[0])
	rawName := strings.TrimSpace(parts[1])
	if rawName == "" || strings.Contains(rawName, "..") || strings.ContainsAny(rawName, `\`) {
		return "", "", errors.New(errInvalidModuleName)
	}
	name := strings.TrimSpace(strings.ReplaceAll(rawName, ".", "/"))
	if (scope != "system" && scope != "business") || name == "" {
		return "", "", errors.New(errInvalidModuleName)
	}
	if !isValidDynamicModulePath(name, scope == "business") {
		return "", "", errors.New(errInvalidModuleName)
	}
	return scope, name, nil
}

func buildModuleKey(scope, name string) string {
	return strings.TrimSpace(scope) + "." + strings.ReplaceAll(strings.Trim(strings.TrimSpace(name), "/"), "/", ".")
}

func resolveGeneratedParentMenu(scope, name, explicitParent string) (string, string) {
	normalizedExplicit := normalizeGeneratedMenuPath(explicitParent)
	if normalizedExplicit != "" {
		return normalizedExplicit, "explicit"
	}
	if scope == "business" {
		segments := strings.Split(strings.Trim(name, "/"), "/")
		if len(segments) > 1 {
			return "/operations/" + segments[0], "inferred"
		}
	}
	return "", "top_level"
}

func inferGeneratedModelName(name, explicit string) string {
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
	moduleName = normalizeStaticModuleName(moduleName)
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

func isValidDynamicModulePath(name string, allowNested bool) bool {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return false
	}
	segments := strings.Split(normalized, "/")
	if !allowNested && len(segments) != 1 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for index, char := range segment {
			if index == 0 {
				if !unicode.IsLower(char) {
					return false
				}
				continue
			}
			if !(unicode.IsLower(char) || unicode.IsDigit(char) || char == '_') {
				return false
			}
		}
	}
	return true
}
