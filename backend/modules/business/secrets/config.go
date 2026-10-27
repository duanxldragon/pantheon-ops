package secrets

import "strings"

// EncryptConfigFields returns a copy whose selected string fields are encrypted.
func EncryptConfigFields(config map[string]interface{}, fields ...string) (map[string]interface{}, error) {
	return transformConfig(config, fieldSet(fields), func(value string) (string, error) {
		if strings.HasPrefix(value, "v1:") {
			return value, nil
		}
		return Encrypt(value)
	})
}

// DecryptConfigFields returns a copy whose selected encrypted fields are plaintext.
func DecryptConfigFields(config map[string]interface{}, fields ...string) (map[string]interface{}, error) {
	return transformConfig(config, fieldSet(fields), func(value string) (string, error) {
		if !strings.HasPrefix(value, "v1:") {
			return value, nil
		}
		return Decrypt(value)
	})
}

// RedactConfigFields removes selected fields and reports whether any were set.
func RedactConfigFields(config map[string]interface{}, fields ...string) map[string]interface{} {
	selected := fieldSet(fields)
	redacted := cloneConfig(config)
	hasSecret := false
	for key, value := range redacted {
		if _, ok := selected[strings.ToLower(key)]; ok {
			if text, isString := value.(string); isString && strings.TrimSpace(text) != "" {
				hasSecret = true
			}
			delete(redacted, key)
		}
	}
	if hasSecret {
		redacted["hasSecret"] = true
	}
	return redacted
}

// PreserveStoredFields keeps encrypted values when an update omits a secret.
func PreserveStoredFields(incoming, stored map[string]interface{}, fields ...string) map[string]interface{} {
	result := cloneConfig(incoming)
	for _, field := range fields {
		value, supplied := result[field]
		text, isString := value.(string)
		if supplied && (!isString || strings.TrimSpace(text) != "") {
			continue
		}
		if storedValue, exists := stored[field]; exists {
			result[field] = storedValue
		}
	}
	return result
}

func transformConfig(config map[string]interface{}, selected map[string]struct{}, transform func(string) (string, error)) (map[string]interface{}, error) {
	result := cloneConfig(config)
	for key, value := range result {
		if _, ok := selected[strings.ToLower(key)]; !ok {
			continue
		}
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		transformed, err := transform(text)
		if err != nil {
			return nil, err
		}
		result[key] = transformed
	}
	return result, nil
}

func cloneConfig(config map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(config))
	for key, value := range config {
		result[key] = value
	}
	return result
}

func fieldSet(fields []string) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[strings.ToLower(strings.TrimSpace(field))] = struct{}{}
	}
	return result
}
