package system

import (
	"strings"
	"testing"
)

// TestUploadContract_SeedDefinesNoUploadPermissionKey locks the upload
// authorization contract (2026-09-22 task): /system/upload is an
// authenticated capability (TokenAuth + TenantContext, no Casbin) used by
// ordinary users for avatars. The permission seed must therefore not
// advertise any upload permission key — a seeded but unenforced key would
// make permission-workbench remediation create dead policies. If the route
// ever moves behind CasbinMiddleware, add the seeded key and this guard in
// the same change.
func TestUploadContract_SeedDefinesNoUploadPermissionKey(t *testing.T) {
	for _, group := range []func() []menuSeed{settingMenuSeeds, baseMenuGroupSeeds, platformToolMenuSeeds} {
		for _, seed := range group() {
			for _, key := range []string{seed.PagePerm, seed.Perms} {
				if strings.Contains(strings.ToLower(strings.TrimSpace(key)), "upload") {
					t.Fatalf("permission seed advertises upload key %q but /system/upload enforces no Casbin policy; align route wiring and seed in one change", key)
				}
			}
		}
	}
}
