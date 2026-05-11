package iam

import (
	"encoding/json"
	"strings"
)

type UserPlatformPreferenceResp struct {
	Theme       string `json:"theme,omitempty"`
	Language    string `json:"language,omitempty"`
	LayoutMode  string `json:"layoutMode,omitempty"`
	DensityMode string `json:"densityMode,omitempty"`
}

var (
	allowedPreferenceThemes = map[string]struct{}{
		"indigo":  {},
		"emerald": {},
		"violet":  {},
		"slate":   {},
	}
	allowedPreferenceLanguages = map[string]struct{}{
		"zh-CN": {},
		"en-US": {},
		"ja-JP": {},
		"ko-KR": {},
		"fr-FR": {},
	}
	allowedPreferenceLayoutModes = map[string]struct{}{
		"vertical":   {},
		"horizontal": {},
	}
	allowedPreferenceDensityModes = map[string]struct{}{
		"comfortable": {},
		"compact":     {},
	}
)

func ParseUserPlatformPreferences(raw string) *UserPlatformPreferenceResp {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	preferences, err := parseRawUserPlatformPreferences(raw)
	if err != nil {
		return nil
	}
	return NormalizeUserPlatformPreferences(preferences)
}

func NormalizeUserPlatformPreferences(preferences *UserPlatformPreferenceResp) *UserPlatformPreferenceResp {
	if preferences == nil {
		return nil
	}

	normalized := &UserPlatformPreferenceResp{
		Theme:       normalizePreferenceValue(preferences.Theme, allowedPreferenceThemes),
		Language:    normalizePreferenceValue(preferences.Language, allowedPreferenceLanguages),
		LayoutMode:  normalizePreferenceValue(preferences.LayoutMode, allowedPreferenceLayoutModes),
		DensityMode: normalizePreferenceValue(preferences.DensityMode, allowedPreferenceDensityModes),
	}
	if normalized.Theme == "" && normalized.Language == "" && normalized.LayoutMode == "" && normalized.DensityMode == "" {
		return nil
	}
	return normalized
}

func MarshalUserPlatformPreferences(preferences *UserPlatformPreferenceResp) (string, error) {
	normalized := NormalizeUserPlatformPreferences(preferences)
	if normalized == nil {
		return "", nil
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizePreferenceValue(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return ""
}

type rawUserPlatformPreferencePayload struct {
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

func parseRawUserPlatformPreferences(raw string) (*UserPlatformPreferenceResp, error) {
	var payload rawUserPlatformPreferencePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}

	return &UserPlatformPreferenceResp{
		Theme:       payload.Theme,
		Language:    firstNonEmptyPreferenceValue(payload.Language, payload.Lang, payload.Locale),
		LayoutMode:  firstNonEmptyPreferenceValue(payload.LayoutMode, payload.Layout, payload.NavigationMode),
		DensityMode: firstNonEmptyPreferenceValue(payload.DensityMode, payload.Density),
	}, nil
}

func firstNonEmptyPreferenceValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
