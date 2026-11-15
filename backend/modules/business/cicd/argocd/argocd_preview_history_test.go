package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewProviderHistoryMatchesExecutableRollback(t *testing.T) {
	const source = `{"repoURL":"https://fixture.invalid/repo.git","path":".","targetRevision":"main","helm":{"values":"private-sentinel"}}`
	provider := &argoApplication{Spec: json.RawMessage(`{"source":` + source + `}`)}
	require.NoError(t, json.Unmarshal([]byte(`{"sync":{"revision":"`+fixtureRevision+`"},"history":[
		{"id":0,"revision":"`+fixtureOldRevision+`","source":`+source+`},
		{"id":9,"revision":"`+fixtureOldRevision+`","source":{"repoURL":"https://fixture.invalid/other.git"}},
		{"id":10,"revision":"`+fixtureRevision+`","source":`+source+`},
		{"id":11,"revision":"token:private-sentinel","source":`+source+`},
		{"id":-1,"revision":"`+fixtureRevision+`","source":`+source+`}
	]}`), &provider.Status))
	history, truncated := previewProviderHistory(provider)
	require.False(t, truncated)
	require.Len(t, history, 2)
	require.True(t, history[0].Current)
	require.Equal(t, int64(0), history[1].ID, "history ID zero is valid and the newer incompatible source is not executable")
	require.True(t, history[1].RollbackAvailable)
	public, err := json.Marshal(history)
	require.NoError(t, err)
	for _, secret := range []string{"private-sentinel", "repoURL", "helm", "values"} {
		require.NotContains(t, string(public), secret)
	}
}

func TestPreviewProviderHistoryBoundsAndUnsupportedSources(t *testing.T) {
	provider := &argoApplication{Spec: json.RawMessage(`{"source":{"repoURL":"https://fixture.invalid/repo.git"}}`)}
	var entries []string
	for i := 0; i < 52; i++ {
		entries = append(entries, fmt.Sprintf(`{"id":%d,"revision":"%040x","sources":[{"repoURL":"hidden"}]}`, i, i))
	}
	require.NoError(t, json.Unmarshal([]byte(`{"history":[`+strings.Join(entries, ",")+`]}`), &provider.Status))
	history, truncated := previewProviderHistory(provider)
	require.True(t, truncated)
	require.Len(t, history, 50)
	require.Equal(t, int64(51), history[0].ID)
	for _, entry := range history {
		require.False(t, entry.RollbackAvailable)
	}
}

func TestApplicationPreviewProjectsProviderHistoryAndExplicitRisks(t *testing.T) {
	s, id := actionService(t, func(http.ResponseWriter, *http.Request) {})
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: fixtureRevision}, productActor())
	require.NoError(t, err)
	require.Equal(t, "test", preview.Namespace)
	require.Len(t, preview.History, 2)
	require.Equal(t, fixtureOldRevision, preview.History[0].Revision)
	require.True(t, preview.History[0].RollbackAvailable)
	require.False(t, preview.RollbackAvailable, "sync preview is not a reviewed rollback")
	require.Contains(t, preview.RiskKeys, "business.cicd.argocd.preview.pruneDisabled")
	require.Contains(t, preview.RiskKeys, "business.cicd.argocd.preview.healthRequired")
	public, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NotContains(t, string(public), "repoURL")
	require.NotContains(t, string(public), "sourceRepoURL")
}
