package argocd

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/datatypes"
)

type syncCall struct {
	appID    uint64
	revision string
}

type recordingSyncer struct {
	calls []syncCall
	err   error
}

func (r *recordingSyncer) SyncApplication(id uint64, revision string) (*SyncHistoryResponse, error) {
	r.calls = append(r.calls, syncCall{appID: id, revision: revision})
	if r.err != nil {
		return nil, r.err
	}
	return &SyncHistoryResponse{}, nil
}

func TestHasAutomatedSyncPolicyOnlyAcceptsArgoCDAutomated(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{name: "empty column", policy: "", want: false},
		{name: "json null", policy: "null", want: false},
		{name: "automated empty object", policy: `{"automated":{}}`, want: true},
		{name: "automated with options", policy: `{"automated":{"prune":true,"selfHeal":true},"syncOptions":["CreateNamespace=true"]}`, want: true},
		{name: "manual sync only", policy: `{"syncOptions":["CreateNamespace=true"]}`, want: false},
		{name: "automated null", policy: `{"automated":null}`, want: false},
		{name: "automated false", policy: `{"automated":false}`, want: false},
		{name: "unparseable column", policy: `{"automated":`, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasAutomatedSyncPolicy([]byte(tc.policy)); got != tc.want {
				t.Fatalf("hasAutomatedSyncPolicy(%q) = %v, want %v", tc.policy, got, tc.want)
			}
		})
	}
}

func TestProcessGitEventSyncsOnlyAutomatedApplications(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ArgoCDApplication{}, &WebhookEvent{}); err != nil {
		t.Fatalf("migrate argocd webhook tables: %v", err)
	}

	const repoURL = "https://git.example.com/team/checkout.git"
	apps := []ArgoCDApplication{
		{Name: "automated", ServerID: 1, SourceRepoURL: repoURL, SourceTargetRev: "HEAD", Status: "Synced",
			SyncPolicy: datatypes.JSON(`{"automated":{"prune":true}}`)},
		{Name: "manual", ServerID: 1, SourceRepoURL: repoURL, Status: "Synced", SyncPolicy: datatypes.JSON(`{}`)},
		{Name: "no policy", ServerID: 1, SourceRepoURL: repoURL, Status: "Synced"},
		{Name: "errored", ServerID: 1, SourceRepoURL: repoURL, Status: "Error", SyncPolicy: datatypes.JSON(`{"automated":{}}`)},
	}
	for i := range apps {
		if err := db.Create(&apps[i]).Error; err != nil {
			t.Fatalf("create application %s: %v", apps[i].Name, err)
		}
	}

	event := WebhookEvent{Source: sourceGitHub, EventType: "push", Repository: repoURL, Payload: "{}"}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create webhook event: %v", err)
	}

	syncer := &recordingSyncer{}
	handler := &WebhookHandler{db: db, syncer: syncer}
	handler.processGitEvent(&event)

	if len(syncer.calls) != 1 {
		t.Fatalf("expected exactly one sync trigger, got %+v", syncer.calls)
	}
	if syncer.calls[0].appID != apps[0].ID || syncer.calls[0].revision != "HEAD" {
		t.Fatalf("unexpected sync target: %+v, want application %d at HEAD", syncer.calls[0], apps[0].ID)
	}

	var stored WebhookEvent
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("reload webhook event: %v", err)
	}
	if !stored.Processed || stored.ProcessedAt == nil {
		t.Fatalf("webhook event was not marked processed: processed=%v processedAt=%v", stored.Processed, stored.ProcessedAt)
	}
}

func TestProcessGitEventKeepsProcessingWhenSyncFails(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ArgoCDApplication{}, &WebhookEvent{}); err != nil {
		t.Fatalf("migrate argocd webhook tables: %v", err)
	}

	const repoURL = "https://git.example.com/team/ledger.git"
	app := ArgoCDApplication{Name: "automated", ServerID: 1, SourceRepoURL: repoURL, Status: "Synced",
		SyncPolicy: datatypes.JSON(`{"automated":{}}`)}
	if err := db.Create(&app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}
	event := WebhookEvent{Source: sourceGitLab, EventType: "push", Repository: repoURL, Payload: "{}"}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create webhook event: %v", err)
	}

	syncer := &recordingSyncer{err: errors.New("argocd unavailable")}
	handler := &WebhookHandler{db: db, syncer: syncer}
	handler.processGitEvent(&event)

	if len(syncer.calls) != 1 {
		t.Fatalf("expected the failing sync to still be attempted, got %+v", syncer.calls)
	}
	var stored WebhookEvent
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("reload webhook event: %v", err)
	}
	if !stored.Processed {
		t.Fatal("a failed sync trigger must not leave the event unprocessed")
	}
}
