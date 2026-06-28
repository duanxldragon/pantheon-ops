package platformprefs

import (
	"encoding/json"
	"strings"
)

type PlatformPreference struct {
	Theme       string `json:"theme,omitempty"`
	Language    string `json:"language,omitempty"`
	LayoutMode  string `json:"layoutMode,omitempty"`
	DensityMode string `json:"densityMode,omitempty"`
}

var (
	allowedThemes = map[string]struct{}{
		"indigo":  {},
		"emerald": {},
		"violet":  {},
		"slate":   {},
	}
	allowedLanguages = map[string]struct{}{
		"zh-CN": {},
		"en-US": {},
		"ja-JP": {},
		"ko-KR": {},
		"fr-FR": {},
	}
	allowedLayoutModes = map[string]struct{}{
		"vertical":   {},
		"horizontal": {},
	}
	allowedDensityModes = map[string]struct{}{
		"comfortable": {},
		"compact":     {},
	}
)

func Parse(raw string) *PlatformPreference {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	preferences, err := parseRaw(raw)
	if err != nil {
		return nil
	}
	return Normalize(preferences)
}

func Normalize(preferences *PlatformPreference) *PlatformPreference {
	if preferences == nil {
		return nil
	}

	normalized := &PlatformPreference{
		Theme:       normalizeValue(preferences.Theme, allowedThemes),
		Language:    normalizeValue(preferences.Language, allowedLanguages),
		LayoutMode:  normalizeValue(preferences.LayoutMode, allowedLayoutModes),
		DensityMode: normalizeValue(preferences.DensityMode, allowedDensityModes),
	}
	if normalized.Theme == "" && normalized.Language == "" && normalized.LayoutMode == "" && normalized.DensityMode == "" {
		return nil
	}
	return normalized
}

func Marshal(preferences *PlatformPreference) (string, error) {
	normalized := Normalize(preferences)
	if normalized == nil {
		return "", nil
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type rawPlatformPreferencePayload struct {
	Theme          string `json:"theme"`
	Language       string `json:"language"`
	LayoutMode     string `json:"layoutMode"`
	DensityMode    string `json:"densityMode"`
	Layout         string `json:"layout"`
	Density        string `json:"density"`
	Lang           string `json:"lang"`
	Locale         string `json:"locale"`
	NavigationMode string `json:"navigationMode"`
}

func parseRaw(raw string) (*PlatformPreference, error) {
	var payload rawPlatformPreferencePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}

	return &PlatformPreference{
		Theme:       payload.Theme,
		Language:    firstNonEmpty(payload.Language, payload.Lang, payload.Locale),
		LayoutMode:  firstNonEmpty(payload.LayoutMode, payload.Layout, payload.NavigationMode),
		DensityMode: firstNonEmpty(payload.DensityMode, payload.Density),
	}, nil
}

func normalizeValue(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
