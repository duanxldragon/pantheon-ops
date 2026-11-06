package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

type sessionRaceResult struct {
	pair *authtoken.Pair
	err  error
}

type refreshRaceLoader struct {
	entered chan struct{}
	release chan struct{}
}

func (l refreshRaceLoader) GetUserByID(id uint64) (*UserRef, error) {
	close(l.entered)
	<-l.release
	return &UserRef{ID: id, Username: "refresh-race"}, nil
}
func (l refreshRaceLoader) GetUserRoles(uint64) ([]string, error) {
	return []string{"tenant_admin"}, nil
}

// Pause after the session read to exercise the actual stale-write window without
// scheduler timing assumptions. A completed revocation or tenant switch must
// survive the refresh that previously read the active row.
func TestRefreshSessionRejectsConcurrentLifecycleChange(t *testing.T) {
	for _, change := range []string{"revocation", "tenant-switch"} {
		t.Run(change, func(t *testing.T) {
			runRefreshSessionLifecycleRace(t, change)
		})
	}
}

func runRefreshSessionLifecycleRace(t *testing.T, change string) {
	t.Helper()
	svc := newSwitchHarness(t)
	if err := svc.db.Model(&SystemUserSession{}).Where("session_id = ?", "sess-switch-1").Update("tenant_id", 202).Error; err != nil {
		t.Fatal(err)
	}
	// The fixture only seeds a membership in tenant 202; compat mode also
	// avoids making the test depend on unrelated tenant gate machinery.
	if err := svc.db.Exec("UPDATE system_setting SET setting_value = ? WHERE setting_key = ?", tenant.ModeCompat, tenant.FeatureFlagSettingKey).Error; err != nil {
		t.Fatal(err)
	}
	previous := database.RDB
	database.RDB = nil
	t.Cleanup(func() { database.RDB = previous })
	entered, release := make(chan struct{}), make(chan struct{})
	svc.loader = refreshRaceLoader{entered, release}
	done := make(chan sessionRaceResult, 1)
	go func() {
		pair, err := svc.RefreshSessionWithContext(context.Background(), "sess-switch-1", 42, "127.0.0.1", "test")
		done <- sessionRaceResult{pair, err}
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("refresh did not reach loader")
	}
	if change == "revocation" {
		if err := svc.RevokeSession("sess-switch-1"); err != nil {
			close(release)
			t.Fatal(err)
		}
	} else {
		if err := svc.db.Model(&SystemUserSession{}).Where("session_id = ?", "sess-switch-1").Updates(map[string]interface{}{"tenant_id": 303, "refresh_jti": "concurrent-switch-jti"}).Error; err != nil {
			close(release)
			t.Fatal(err)
		}
	}
	close(release)
	res := <-done
	var stored SystemUserSession
	if err := svc.db.Where("session_id = ?", "sess-switch-1").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	assertRefreshLifecycleRace(t, change, res, stored)
}

func TestSwitchSessionRejectsConcurrentLifecycleChange(t *testing.T) {
	for _, change := range []string{"revocation", "rotation"} {
		t.Run(change, func(t *testing.T) {
			runSwitchSessionLifecycleRace(t, change)
		})
	}
}

func runSwitchSessionLifecycleRace(t *testing.T, change string) {
	t.Helper()
	svc := newSwitchHarness(t)
	previous := database.RDB
	database.RDB = nil
	t.Cleanup(func() { database.RDB = previous })
	entered, release := make(chan struct{}), make(chan struct{})
	svc.loader = refreshRaceLoader{entered, release}
	done := make(chan sessionRaceResult, 1)
	go func() {
		pair, err := svc.SwitchSessionTenant(context.Background(), "sess-switch-1", 42, 202)
		done <- sessionRaceResult{pair, err}
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("switch did not reach loader")
	}
	if change == "revocation" {
		if err := svc.RevokeSession("sess-switch-1"); err != nil {
			close(release)
			t.Fatal(err)
		}
	} else {
		if err := svc.db.Model(&SystemUserSession{}).Where("session_id = ?", "sess-switch-1").Update("refresh_jti", "concurrent-rotation-jti").Error; err != nil {
			close(release)
			t.Fatal(err)
		}
	}
	close(release)
	res := <-done
	if !errors.Is(res.err, common.ErrUnauthorized) || res.pair != nil {
		t.Errorf("stale switch pair=%+v err=%v", res.pair, res.err)
	}
	var stored SystemUserSession
	if err := svc.db.Where("session_id = ?", "sess-switch-1").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TenantID != 0 {
		t.Error("stale switch changed tenant")
	}
	if change == "revocation" && stored.RevokedAt == nil {
		t.Error("switch resurrected revoked session")
	}
	if change == "rotation" && stored.RefreshJTI != "concurrent-rotation-jti" {
		t.Error("switch overwrote concurrent rotation")
	}
}

func assertRefreshLifecycleRace(t *testing.T, change string, res sessionRaceResult, stored SystemUserSession) {
	t.Helper()
	t.Logf("change=%s pairIssued=%t revoked=%t tenant=%d", change, res.pair != nil, stored.RevokedAt != nil, stored.TenantID)
	if !errors.Is(res.err, common.ErrUnauthorized) || res.pair != nil {
		t.Errorf("stale refresh pair=%+v err=%v; want unauthorized and no pair", res.pair, res.err)
	}
	if change == "revocation" && stored.RevokedAt == nil {
		t.Error("refresh resurrected revoked session")
	}
	if change == "tenant-switch" && (stored.TenantID != 303 || stored.RefreshJTI != "concurrent-switch-jti") {
		t.Error("refresh overwrote concurrent tenant switch")
	}
}
