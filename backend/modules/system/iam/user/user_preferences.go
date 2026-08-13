package iam

import "pantheon-base/pkg/platformprefs"

type UserPlatformPreferenceResp = platformprefs.PlatformPreference

func ParseUserPlatformPreferences(raw string) *UserPlatformPreferenceResp {
	return platformprefs.Parse(raw)
}

func NormalizeUserPlatformPreferences(preferences *UserPlatformPreferenceResp) *UserPlatformPreferenceResp {
	return platformprefs.Normalize(preferences)
}

func MarshalUserPlatformPreferences(preferences *UserPlatformPreferenceResp) (string, error) {
	return platformprefs.Marshal(preferences)
}
