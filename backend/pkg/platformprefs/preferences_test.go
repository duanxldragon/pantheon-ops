package platformprefs

import "testing"

func TestParseNormalizesLegacyPlatformPreferencePayload(t *testing.T) {
	preferences := Parse(`{"theme":"emerald","layout":"horizontal","density":"compact","lang":"en-US","extra":"ignored"}`)
	if preferences == nil {
		t.Fatalf("expected normalized preferences")
	}
	if preferences.Theme != "emerald" || preferences.Language != "en-US" || preferences.LayoutMode != "horizontal" || preferences.DensityMode != "compact" {
		t.Fatalf("unexpected normalized preferences: %+v", preferences)
	}
}

func TestMarshalReturnsCanonicalPlatformPreferenceJSON(t *testing.T) {
	payload, err := Marshal(&PlatformPreference{
		Theme:       "slate",
		Language:    "en-US",
		LayoutMode:  "horizontal",
		DensityMode: "compact",
	})
	if err != nil {
		t.Fatalf("marshal preferences: %v", err)
	}
	if payload != `{"theme":"slate","language":"en-US","layoutMode":"horizontal","densityMode":"compact"}` {
		t.Fatalf("unexpected canonical preference json: %s", payload)
	}
}

func TestNormalizeDropsUnsupportedPlatformPreferenceValues(t *testing.T) {
	preferences := Normalize(&PlatformPreference{
		Theme:       "invalid",
		Language:    "de-DE",
		LayoutMode:  "grid",
		DensityMode: "wide",
	})
	if preferences != nil {
		t.Fatalf("expected invalid preferences to normalize to nil, got %+v", preferences)
	}
}
