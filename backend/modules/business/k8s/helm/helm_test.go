package helm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
)

// TestCreateForcedMisleadingStatusIsRegistrationOnly verifies that a caller
// cannot make a registration row claim a Helm runtime state through Create:
// empty, forced "deployed" and arbitrary runtime-looking statuses all persist
// and return registeredStatus only.
func TestCreateForcedMisleadingStatusIsRegistrationOnly(t *testing.T) {
	cases := []struct {
		name      string
		submitted string
	}{
		{name: "empty", submitted: ""},
		{name: "deployed", submitted: "deployed"},
		{name: "runtime-looking", submitted: "pending-upgrade"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testmysql.Open(t)
			if err := db.AutoMigrate(&HelmRelease{}); err != nil {
				t.Fatalf("migrate helm release: %v", err)
			}
			svc := NewHelmService(db)

			rec, err := svc.Create(CreateHelmReleaseRequest{
				Name:      "redis",
				ClusterID: 1,
				Namespace: "prod",
				Chart:     "redis",
				Version:   "1.0.0",
				Status:    tc.submitted,
			}, "1")
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if rec.Status != registeredStatus {
				t.Fatalf("returned status = %q; want %q (submitted %q ignored)", rec.Status, registeredStatus, tc.submitted)
			}
			if rec.EffectiveStatus != registeredStatus {
				t.Fatalf("returned effectiveStatus = %q; want %q", rec.EffectiveStatus, registeredStatus)
			}
			if rec.StoredStatus != registeredStatus {
				t.Fatalf("returned storedStatus = %q; want %q", rec.StoredStatus, registeredStatus)
			}

			var persisted HelmRelease
			if err := db.First(&persisted, rec.ID).Error; err != nil {
				t.Fatalf("reload persisted: %v", err)
			}
			if persisted.Status != registeredStatus {
				t.Fatalf("persisted status = %q; want %q (submitted %q must not be recorded)",
					persisted.Status, registeredStatus, tc.submitted)
			}
			if persisted.EffectiveStatus != "" {
				t.Fatalf("effectiveStatus must be a response-only field, got persisted value %q", persisted.EffectiveStatus)
			}
			if persisted.StoredStatus != "" {
				t.Fatalf("storedStatus must be a response-only field, got persisted value %q", persisted.StoredStatus)
			}
		})
	}
}

// TestListResponseNormalizesLegacyStoredStatus verifies that legacy DB rows
// carrying a historical stored value ("deployed"/"failed") keep that raw
// provenance untouched while the API exposes a registration-only effective
// status, and that a DB row alone is never presented as runtime deployment
// proof.
func TestListResponseNormalizesLegacyStoredStatus(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&HelmRelease{}); err != nil {
		t.Fatalf("migrate helm release: %v", err)
	}
	svc := NewHelmService(db)

	// Legacy rows persisted before the correction keep their stored values.
	legacyDeployed := HelmRelease{Name: "legacy-deployed", ClusterID: 1, Namespace: "prod", Chart: "redis", Status: "deployed"}
	if err := db.Create(&legacyDeployed).Error; err != nil {
		t.Fatalf("create legacy deployed row: %v", err)
	}
	legacyOther := HelmRelease{Name: "legacy-other", ClusterID: 1, Namespace: "prod", Chart: "redis", Status: "failed"}
	if err := db.Create(&legacyOther).Error; err != nil {
		t.Fatalf("create legacy other row: %v", err)
	}
	// A corrected registration.
	if _, err := svc.Create(CreateHelmReleaseRequest{Name: "new", ClusterID: 1, Namespace: "prod", Chart: "redis", Version: "1.0.0"}, "1"); err != nil {
		t.Fatalf("create new registration: %v", err)
	}

	resp, err := svc.List(ListQuery{}, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("total = %d; want 3", resp.Total)
	}
	got := make(map[string]HelmRelease, len(resp.Items))
	for _, item := range resp.Items {
		got[item.Name] = item
	}

	for _, name := range []string{"legacy-deployed", "legacy-other", "new"} {
		item, ok := got[name]
		if !ok {
			t.Fatalf("missing %q in list response", name)
		}
		if item.EffectiveStatus != registeredStatus {
			t.Fatalf("%q effectiveStatus = %q; want registration-only %q", name, item.EffectiveStatus, registeredStatus)
		}
		if item.Status != registeredStatus {
			t.Fatalf("%q status = %q; want registration-only %q", name, item.Status, registeredStatus)
		}
	}
	if got["legacy-deployed"].StoredStatus != "deployed" {
		t.Fatalf("legacy stored provenance = %q; want preserved %q", got["legacy-deployed"].StoredStatus, "deployed")
	}
	if got["legacy-other"].StoredStatus != "failed" {
		t.Fatalf("legacy stored provenance = %q; want preserved %q", got["legacy-other"].StoredStatus, "failed")
	}
	if got["new"].StoredStatus != registeredStatus {
		t.Fatalf("new stored provenance = %q; want %q", got["new"].StoredStatus, registeredStatus)
	}

	// No destructive bulk rewrite in the DB.
	var stored []HelmRelease
	if err := db.Order("id ASC").Find(&stored).Error; err != nil {
		t.Fatalf("reload stored rows: %v", err)
	}
	storedBy := make(map[string]string, len(stored))
	for _, row := range stored {
		storedBy[row.Name] = row.Status
	}
	if storedBy["legacy-deployed"] != "deployed" {
		t.Fatalf("DB legacy row rewritten to %q; want preserved %q", storedBy["legacy-deployed"], "deployed")
	}
	if storedBy["legacy-other"] != "failed" {
		t.Fatalf("DB legacy row rewritten to %q; want preserved %q", storedBy["legacy-other"], "failed")
	}
	if storedBy["new"] != registeredStatus {
		t.Fatalf("DB new row status = %q; want %q", storedBy["new"], registeredStatus)
	}
}

// TestHelmRegistrationCapabilityBoundaryHTTP drills the HTTP boundary: a
// forced "deployed" create stays registration-only in the JSON response and a
// legacy stored "deployed" row is returned with effectiveStatus registered.
func TestHelmRegistrationCapabilityBoundaryHTTP(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&HelmRelease{}); err != nil {
		t.Fatalf("migrate helm release: %v", err)
	}
	svc := NewHelmService(db)
	if err := db.Create(&HelmRelease{Name: "legacy", ClusterID: 1, Namespace: "prod", Status: "deployed"}).Error; err != nil {
		t.Fatalf("create legacy row: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/business/k8s")
	NewHelmHandler(svc).RegisterRoutes(group)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/business/k8s/helm/releases",
		strings.NewReader(`{"name":"redis","clusterId":1,"namespace":"prod","chart":"redis","version":"1.0.0","status":"deployed"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	var createEnvelope struct {
		Code int         `json:"code"`
		Data HelmRelease `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createEnvelope); err != nil {
		t.Fatalf("unmarshal create response: %v (%s)", err, createRec.Body.String())
	}
	if createRec.Code != http.StatusOK || createEnvelope.Code != common.CodeSuccess || createEnvelope.Data.ID == 0 {
		t.Fatalf("create did not return a successful persisted registration: HTTP %d, code %d, ID %d", createRec.Code, createEnvelope.Code, createEnvelope.Data.ID)
	}
	if createEnvelope.Data.Status != registeredStatus {
		t.Fatalf("create status = %q; want %q, forced deployed must not escape", createEnvelope.Data.Status, registeredStatus)
	}
	if createEnvelope.Data.EffectiveStatus != registeredStatus {
		t.Fatalf("create effectiveStatus = %q; want %q", createEnvelope.Data.EffectiveStatus, registeredStatus)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/business/k8s/helm/releases", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	var listEnvelope struct {
		Code int          `json:"code"`
		Data ListResponse `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("unmarshal list response: %v (%s)", err, listRec.Body.String())
	}
	if listRec.Code != http.StatusOK || listEnvelope.Code != common.CodeSuccess || listEnvelope.Data.Total != 2 || len(listEnvelope.Data.Items) != 2 {
		t.Fatalf("list did not return both registrations successfully: HTTP %d, code %d, total %d, items %d", listRec.Code, listEnvelope.Code, listEnvelope.Data.Total, len(listEnvelope.Data.Items))
	}
	items := make(map[string]HelmRelease, len(listEnvelope.Data.Items))
	for _, item := range listEnvelope.Data.Items {
		items[item.Name] = item
	}
	created, ok := items["redis"]
	if !ok || created.ID != createEnvelope.Data.ID || created.Status != registeredStatus || created.EffectiveStatus != registeredStatus {
		t.Fatalf("created registration changed or disappeared in list: %+v", created)
	}
	legacy, ok := items["legacy"]
	if !ok {
		t.Fatalf("legacy row missing from list; got %d items", len(items))
	}
	if legacy.EffectiveStatus != registeredStatus {
		t.Fatalf("legacy effectiveStatus = %q; want registration-only %q", legacy.EffectiveStatus, registeredStatus)
	}
	// The historical stored value is explicitly preserved as provenance, not
	// presented as an effective runtime state.
	if legacy.Status != registeredStatus || legacy.StoredStatus != "deployed" {
		t.Fatalf("legacy response status/provenance = %q/%q; want %q/%q", legacy.Status, legacy.StoredStatus, registeredStatus, "deployed")
	}
}

func TestHelmResponsesProjectRegistrationStatusAndPreserveStoredStatus(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&HelmRelease{}); err != nil {
		t.Fatal(err)
	}
	legacy := HelmRelease{Name: "legacy", ClusterID: 1, Namespace: "prod", Status: "deployed"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHelmHandler(NewHelmService(db)).RegisterRoutes(router.Group("/api/v1/business/k8s"))
	type responseState struct {
		Name            string `json:"name"`
		Status          string `json:"status"`
		StoredStatus    string `json:"storedStatus"`
		EffectiveStatus string `json:"effectiveStatus"`
	}
	t.Run("create", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/business/k8s/helm/releases", strings.NewReader(`{"name":"new","clusterId":1,"namespace":"prod","status":"deployed"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct {
			Code int           `json:"code"`
			Data responseState `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Code != common.CodeSuccess || envelope.Data.Status != registeredStatus || envelope.Data.EffectiveStatus != registeredStatus || envelope.Data.StoredStatus != registeredStatus {
			t.Fatalf("create registration projection: %+v", envelope)
		}
	})
	t.Run("legacy-list", func(t *testing.T) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/business/k8s/helm/releases", nil))
		var envelope struct {
			Code int `json:"code"`
			Data struct {
				Items []responseState `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Code != common.CodeSuccess {
			t.Fatalf("list failed: code %d", envelope.Code)
		}
		for _, item := range envelope.Data.Items {
			if item.Name == legacy.Name {
				if item.Status != registeredStatus || item.EffectiveStatus != registeredStatus || item.StoredStatus != "deployed" {
					t.Fatalf("legacy registration projection: %+v", item)
				}
				return
			}
		}
		t.Fatal("legacy registration missing")
	})
	var persisted HelmRelease
	if err := db.First(&persisted, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Status != "deployed" {
		t.Fatalf("legacy DB provenance was rewritten: %q", persisted.Status)
	}
}
