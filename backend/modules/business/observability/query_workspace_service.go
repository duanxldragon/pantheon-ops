package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrWorkspaceDenied is returned when the workspace owner lacks permission.
var ErrWorkspaceDenied = errors.New("permission.denied")

// ErrWorkspaceQuota is returned when a workspace budget is exceeded.
var ErrWorkspaceQuota = errors.New("observability.workspace_quota")

// ErrWorkspaceConflict is returned on a concurrent workspace write conflict.
var ErrWorkspaceConflict = errors.New("observability.workspace_conflict")

// QueryOwner identifies the tenant and actor that owns a query workspace.
type QueryOwner struct{ TenantID, ActorID uint64 }

// QueryWorkspaceService manages query history and saved views for one owner.
type QueryWorkspaceService struct {
	db    *gorm.DB
	owner QueryOwner
	scope *common.DataScopeReq
}

// NewQueryWorkspaceService binds the workspace service to a database and owner.
func NewQueryWorkspaceService(db *gorm.DB, owner QueryOwner, scope *common.DataScopeReq) (*QueryWorkspaceService, error) {
	if db == nil || owner.ActorID == 0 || scope == nil {
		return nil, ErrWorkspaceDenied
	}
	return &QueryWorkspaceService{db: db, owner: owner, scope: scope}, nil
}
func ownerPredicate(db *gorm.DB, owner QueryOwner) *gorm.DB {
	return db.Where("tenant_id = ? AND actor_id = ?", owner.TenantID, owner.ActorID)
}
func lockQueryOwners(tx *gorm.DB, owners []QueryOwner) error {
	owners = append([]QueryOwner(nil), owners...)
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].TenantID != owners[j].TenantID {
			return owners[i].TenantID < owners[j].TenantID
		}
		return owners[i].ActorID < owners[j].ActorID
	})
	var previous *QueryOwner
	for _, owner := range owners {
		if previous != nil && *previous == owner {
			continue
		}
		copied := owner
		previous = &copied
		guard := QueryOwnerBudget{TenantID: owner.TenantID, ActorID: owner.ActorID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&guard).Error; err != nil {
			return err
		}
		if err := ownerPredicate(tx, owner).Clauses(clause.Locking{Strength: "UPDATE"}).First(&guard).Error; err != nil {
			return err
		}
	}
	return nil
}
func (s *QueryWorkspaceService) source(tx *gorm.DB, id uint64, lock bool) (*PrometheusDatasource, error) {
	q := tx.Where("tenant_id = ?", s.owner.TenantID).Scopes(database.WithDataScope(s.scope))
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var source PrometheusDatasource
	if err := q.First(&source, id).Error; err != nil {
		return nil, ErrWorkspaceDenied
	}
	return &source, nil
}
func (s *QueryWorkspaceService) ownedTransaction(ctx context.Context, sourceIDs []uint64, fn func(*gorm.DB) error) error {
	ids := append([]uint64(nil), sourceIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockQueryOwners(tx, []QueryOwner{s.owner}); err != nil {
			return err
		}
		var last uint64
		for _, id := range ids {
			if id == 0 || id == last {
				continue
			}
			last = id
			var source PrometheusDatasource
			// Deletion of one's own revoked-source metadata needs no source permission.
			if err := tx.Where("tenant_id = ?", s.owner.TenantID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, id).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return fn(tx)
	})
}

func historySourceIDs(tx *gorm.DB, owner QueryOwner, sourceID uint64, locking bool) ([]uint64, error) {
	var rows []struct{ DatasourceID uint64 }
	q := ownerPredicate(tx.Model(&QueryHistory{}), owner).Select("datasource_id").Order("id")
	if locking {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := []uint64{sourceID}
	for _, row := range rows {
		ids = append(ids, row.DatasourceID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	unique := ids[:0]
	for _, id := range ids {
		if len(unique) == 0 || unique[len(unique)-1] != id {
			unique = append(unique, id)
		}
	}
	return unique, nil
}

func (s *QueryWorkspaceService) historyTransaction(ctx context.Context, sourceID uint64, fn func(*gorm.DB) error) error {
	// Appending may evict histories on other sources. Enumerate all affected
	// sources before taking the owner guard, then verify with a current locking
	// read. Never add source locks after history row locks have been acquired.
	for attempt := 0; attempt < 3; attempt++ {
		ids, err := historySourceIDs(s.db.WithContext(ctx), s.owner, sourceID, false)
		if err != nil {
			return err
		}
		err = s.ownedTransaction(ctx, ids, func(tx *gorm.DB) error {
			current, err := historySourceIDs(tx, s.owner, sourceID, true)
			if err != nil {
				return err
			}
			if !equalIDs(ids, current) {
				return ErrWorkspaceConflict
			}
			return fn(tx)
		})
		if !errors.Is(err, ErrWorkspaceConflict) {
			return err
		}
	}
	return ErrWorkspaceConflict
}

// Save persists or concurrency-guarded updates a saved view for a query recipe.
func (s *QueryWorkspaceService) Save(ctx context.Context, sourceID, id, expectedRevision uint64, recipe QueryRecipe) (*SavedView, error) {
	var result SavedView
	err := s.ownedTransaction(ctx, []uint64{sourceID}, func(tx *gorm.DB) error {
		source, err := s.source(tx, sourceID, false)
		if err != nil {
			return err
		}
		compiled, err := CompileQueryRecipe(recipe, source.Type, time.Now().UTC())
		if err != nil {
			return err
		}
		if id != 0 {
			if err := ownerPredicate(tx, s.owner).Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, id).Error; err != nil {
				return ErrWorkspaceDenied
			}
			if result.DatasourceID != sourceID || result.Revision != expectedRevision || expectedRevision == 0 {
				return ErrWorkspaceConflict
			}
		}
		var stats struct {
			Count int64
			Bytes int64
		}
		if err := ownerPredicate(tx.Model(&SavedView{}), s.owner).Where("id <> ?", id).Select("COUNT(*) AS count, COALESCE(SUM(OCTET_LENGTH(recipe)),0) AS bytes").Scan(&stats).Error; err != nil {
			return err
		}
		if stats.Count >= 50 || stats.Bytes+int64(len(compiled.Canonical)) > 256<<10 {
			return ErrWorkspaceQuota
		}
		if id == 0 {
			result = SavedView{TenantID: s.owner.TenantID, ActorID: s.owner.ActorID, DatasourceID: sourceID, Recipe: datatypes.JSON(compiled.Canonical), Revision: 1}
			return tx.Create(&result).Error
		}
		result.Recipe = datatypes.JSON(compiled.Canonical)
		result.Revision++
		return ownerPredicate(tx.Model(&SavedView{}), s.owner).Where("id = ? AND revision = ?", id, expectedRevision).Updates(map[string]any{"recipe": result.Recipe, "revision": result.Revision, "updated_at": time.Now().UTC()}).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HistoryInput carries the validated fields for a workspace history append.
type HistoryInput struct {
	HistoryMode    string
	HistoryID      uint64
	DatasourceID   uint64
	SourceRevision uint64
	SourceKind     string
	Time           QueryTime
	Limit          int
	ResultMode     string
	Outcome        string
	DurationMS     uint32
	Recipe         *QueryRecipe
}

// AppendHistory records a workspace query entry after validating its intent.
func (s *QueryWorkspaceService) AppendHistory(ctx context.Context, input HistoryInput) (*QueryHistory, error) {
	if input.HistoryMode == "" {
		input.HistoryMode = historyModeRecord
	}
	if (input.HistoryMode != historyModeRecord && input.HistoryMode != historyModeCoalesced) || (input.HistoryMode == historyModeRecord && input.HistoryID != 0) {
		return nil, ErrQueryIntent
	}
	switch input.SourceKind {
	case sourceKindPrometheus, sourceKindVictoriaMetrics, "loki", sourceKindTempo:
	default:
		return nil, ErrQueryIntent
	}
	if input.DurationMS > 30000 {
		return nil, ErrQueryIntent
	}
	switch input.Outcome {
	case queryStatusSuccess, "denied", queryOutcomeInvalid, queryOutcomeUpstream, "storage", queryOutcomeCancelled:
	default:
		return nil, ErrQueryIntent
	}
	if input.ResultMode != resultModeTable || input.Limit <= 0 {
		return nil, ErrQueryIntent
	}
	if _, _, err := queryTimeBounds(input.Time, time.Now().UTC()); err != nil {
		return nil, err
	}
	var result QueryHistory
	err := s.historyTransaction(ctx, input.DatasourceID, func(tx *gorm.DB) error {
		source, err := s.source(tx, input.DatasourceID, false)
		if err != nil {
			return err
		}
		if source.Revision != input.SourceRevision || source.Type != input.SourceKind {
			return ErrWorkspaceConflict
		}
		max := 1000
		if source.Type == "loki" {
			max = 5000
		}
		if input.Limit > max {
			return ErrQueryIntent
		}
		var canonical []byte
		if input.Recipe != nil {
			compiled, err := CompileQueryRecipe(*input.Recipe, source.Type, time.Now().UTC())
			if err != nil {
				return err
			}
			canonical = compiled.Canonical
			if input.Recipe.Time != input.Time || input.Recipe.Limit != input.Limit || input.Recipe.ResultMode != input.ResultMode {
				return ErrQueryIntent
			}
		}
		timeBytes, err := json.Marshal(input.Time)
		if err != nil {
			return ErrQueryIntent
		}
		result = QueryHistory{TenantID: s.owner.TenantID, ActorID: s.owner.ActorID, DatasourceID: source.ID, SourceKind: source.Type, Time: datatypes.JSON(timeBytes), Limit: input.Limit, ResultMode: input.ResultMode, Outcome: input.Outcome, DurationMS: input.DurationMS, Recipe: datatypes.JSON(canonical), ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour)}
		if input.HistoryMode == historyModeCoalesced && input.HistoryID != 0 {
			var previous QueryHistory
			if err := ownerPredicate(tx, s.owner).Clauses(clause.Locking{Strength: "UPDATE"}).First(&previous, input.HistoryID).Error; err != nil {
				return ErrWorkspaceDenied
			}
			// Compare catalog intent and safe metadata only; editor expressions
			// never participate in persistence or session identity.
			var previousTime QueryTime
			var previousCanonical []byte
			if len(previous.Recipe) > 0 {
				previousRecipe, err := DecodeQueryRecipe(bytes.NewReader(previous.Recipe))
				if err != nil {
					return ErrQueryIntent
				}
				compiled, err := CompileQueryRecipe(previousRecipe, source.Type, time.Now().UTC())
				if err != nil {
					return err
				}
				previousCanonical = compiled.Canonical
			}
			if strictQueryJSON(bytes.NewReader(previous.Time), &previousTime) != nil || previous.DatasourceID != source.ID || previous.SourceKind != source.Type || previousTime != input.Time || previous.Limit != input.Limit || previous.ResultMode != input.ResultMode || !bytes.Equal(previousCanonical, canonical) || !previous.ExpiresAt.After(time.Now().UTC()) {
				return ErrWorkspaceConflict
			}
			result.ID, result.CreatedAt = previous.ID, previous.CreatedAt
			return ownerPredicate(tx.Model(&QueryHistory{}), s.owner).Where("id = ?", previous.ID).Updates(map[string]any{"outcome": input.Outcome, "duration_ms": input.DurationMS, "expires_at": result.ExpiresAt}).Error
		}
		if err := ownerPredicate(tx, s.owner).Where("expires_at <= ?", time.Now().UTC()).Delete(&QueryHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		var rows []struct {
			ID    uint64
			Bytes int64
		}
		if err := ownerPredicate(tx.Model(&QueryHistory{}), s.owner).Select("id, COALESCE(OCTET_LENGTH(recipe),0)+OCTET_LENGTH(time_intent) AS bytes").Order("created_at DESC,id DESC").Find(&rows).Error; err != nil {
			return err
		}
		var total int64
		var remove []uint64
		for i, row := range rows {
			total += row.Bytes
			if i >= 500 || total > 2<<20 {
				remove = append(remove, row.ID)
			}
		}
		if len(remove) > 0 {
			return ownerPredicate(tx, s.owner).Where("id IN ?", remove).Delete(&QueryHistory{}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *QueryWorkspaceService) visibleSources(tx *gorm.DB) *gorm.DB {
	return tx.Model(&PrometheusDatasource{}).Where("tenant_id = ?", s.owner.TenantID).Scopes(database.WithDataScope(s.scope)).Select("id")
}
func boundedPage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// History returns the owner's query history rows matching the given id.
func (s *QueryWorkspaceService) History(ctx context.Context, id uint64, page, size int) ([]QueryHistory, int64, error) {
	page, size = boundedPage(page, size)
	tx := s.db.WithContext(ctx)
	q := ownerPredicate(tx.Model(&QueryHistory{}), s.owner).Where("datasource_id IN (?) AND expires_at > ?", s.visibleSources(tx), time.Now().UTC())
	if id != 0 {
		q = q.Where("id = ?", id)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []QueryHistory
	if err := q.Order("created_at DESC,id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	for _, row := range rows {
		source, err := s.source(tx, row.DatasourceID, false)
		if err != nil || source.Type != row.SourceKind {
			return nil, 0, ErrWorkspaceDenied
		}
		if err := validateStoredHistory(row, source.Type); err != nil {
			return nil, 0, err
		}
		if len(row.Recipe) > 0 {
			recipe, err := DecodeQueryRecipe(bytes.NewReader(row.Recipe))
			if err != nil {
				return nil, 0, err
			}
			if _, err = CompileQueryRecipe(recipe, source.Type, time.Now().UTC()); err != nil {
				return nil, 0, err
			}
		}
	}
	if id != 0 && len(rows) == 0 {
		return nil, 0, ErrWorkspaceDenied
	}
	return rows, total, nil
}

func validateStoredHistory(row QueryHistory, kind string) error {
	catalog, err := queryCatalog(kind)
	if err != nil {
		return err
	}
	if row.DurationMS > 30000 || row.Limit <= 0 || row.Limit > catalog.MaxLimit || row.ResultMode != resultModeTable {
		return ErrQueryIntent
	}
	switch row.Outcome {
	case queryStatusSuccess, "denied", queryOutcomeInvalid, queryOutcomeUpstream, "storage", queryOutcomeCancelled:
	default:
		return ErrQueryIntent
	}
	var intent QueryTime
	if err := strictQueryJSON(bytes.NewReader(row.Time), &intent); err != nil {
		return err
	}
	if _, _, err := queryTimeBounds(intent, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

// Views returns the owner's saved views matching the given id.
func (s *QueryWorkspaceService) Views(ctx context.Context, id uint64, page, size int) ([]SavedView, int64, error) {
	page, size = boundedPage(page, size)
	tx := s.db.WithContext(ctx)
	q := ownerPredicate(tx.Model(&SavedView{}), s.owner).Where("datasource_id IN (?)", s.visibleSources(tx))
	if id != 0 {
		q = q.Where("id = ?", id)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SavedView
	if err := q.Order("updated_at DESC,id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	for _, row := range rows {
		source, err := s.source(tx, row.DatasourceID, false)
		if err != nil {
			return nil, 0, err
		}
		recipe, err := DecodeQueryRecipe(bytes.NewReader(row.Recipe))
		if err != nil {
			return nil, 0, err
		}
		if _, err = CompileQueryRecipe(recipe, source.Type, time.Now().UTC()); err != nil {
			return nil, 0, err
		}
	}
	if id != 0 && len(rows) == 0 {
		return nil, 0, ErrWorkspaceDenied
	}
	return rows, total, nil
}

// Delete removes a saved view or query history entry owned by this owner.
func (s *QueryWorkspaceService) Delete(ctx context.Context, view bool, id uint64) error {
	if id == 0 && view {
		return ErrQueryIntent
	}
	for attempt := 0; attempt < 3; attempt++ {
		var sources []uint64
		q := ownerPredicate(s.db.WithContext(ctx), s.owner)
		if id != 0 {
			q = q.Where("id = ?", id)
		}
		model := any(&QueryHistory{})
		if view {
			model = &SavedView{}
		}
		if err := q.Model(model).Distinct("datasource_id").Pluck("datasource_id", &sources).Error; err != nil {
			return err
		}
		err := s.ownedTransaction(ctx, sources, func(tx *gorm.DB) error {
			var current []uint64
			q := ownerPredicate(tx, s.owner)
			if id != 0 {
				q = q.Where("id = ?", id)
			}
			if err := q.Model(model).Clauses(clause.Locking{Strength: "UPDATE"}).Distinct("datasource_id").Pluck("datasource_id", &current).Error; err != nil {
				return err
			}
			sort.Slice(current, func(i, j int) bool { return current[i] < current[j] })
			sort.Slice(sources, func(i, j int) bool { return sources[i] < sources[j] })
			if !equalIDs(current, sources) {
				return ErrWorkspaceConflict
			}
			return q.Delete(model).Error
		})
		if !errors.Is(err, ErrWorkspaceConflict) {
			return err
		}
	}
	return ErrWorkspaceConflict
}
func equalIDs(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
