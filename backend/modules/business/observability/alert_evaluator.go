package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"pantheon-base/modules/business/secrets"

	"gorm.io/gorm"
)

// AlertEvaluator evaluates alert rules periodically.
type AlertEvaluator struct {
	db                  *gorm.DB
	datasourceService   *DatasourceService
	alertRuleService    *AlertRuleService
	notificationService *NotificationService
	evaluators          map[uint64]*ruleEvaluator
	mu                  sync.RWMutex
	stopCh              chan struct{}
	stopOnce            sync.Once
}

type ruleEvaluator struct {
	rule     *AlertRule
	ticker   *time.Ticker
	state    *alertState
	stopCh   chan struct{}
}

type alertState struct {
	firing       bool
	notified     bool
	firingSince  time.Time
	lastValue    float64
	consecutiveOK int
}

// NewAlertEvaluator creates a new AlertEvaluator.
func NewAlertEvaluator(db *gorm.DB, datasourceService *DatasourceService, alertRuleService *AlertRuleService, notificationService *NotificationService) *AlertEvaluator {
	return &AlertEvaluator{
		db:                  db,
		datasourceService:   datasourceService,
		alertRuleService:    alertRuleService,
		notificationService: notificationService,
		evaluators:          make(map[uint64]*ruleEvaluator),
		stopCh:              make(chan struct{}),
	}
}

// Start begins evaluating all enabled alert rules.
func (e *AlertEvaluator) Start(ctx context.Context) {
	// Load all enabled rules
	var rules []AlertRule
	if err := e.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return
	}

	// Start evaluator for each rule
	for _, rule := range rules {
		e.AddRule(&rule)
	}

	// Watch for context cancellation
	go func() {
		<-ctx.Done()
		e.Stop()
	}()
}

// Stop stops all rule evaluators.
func (e *AlertEvaluator) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopCh)
		e.mu.Lock()
		defer e.mu.Unlock()
		for _, ev := range e.evaluators {
			ev.ticker.Stop()
			close(ev.stopCh)
		}
		e.evaluators = make(map[uint64]*ruleEvaluator)
	})
}

// AddRule starts evaluating a new rule.
func (e *AlertEvaluator) AddRule(rule *AlertRule) {
	if !rule.Enabled {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing evaluator if any
	if existing, ok := e.evaluators[rule.ID]; ok {
		existing.ticker.Stop()
		close(existing.stopCh)
	}

	// Create new evaluator
	ev := &ruleEvaluator{
		rule:   rule,
		ticker: time.NewTicker(time.Duration(rule.Interval) * time.Second),
		state:  &alertState{},
		stopCh: make(chan struct{}),
	}

	e.evaluators[rule.ID] = ev

	// Start evaluation loop
	go e.evaluateLoop(ev)
}

// RemoveRule stops evaluating a rule.
func (e *AlertEvaluator) RemoveRule(ruleID uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ev, ok := e.evaluators[ruleID]; ok {
		ev.ticker.Stop()
		close(ev.stopCh)
		delete(e.evaluators, ruleID)
	}
}

func (e *AlertEvaluator) evaluateLoop(ev *ruleEvaluator) {
	for {
		select {
		case <-ev.ticker.C:
			e.evaluateRule(ev)
		case <-ev.stopCh:
			return
		case <-e.stopCh:
			return
		}
	}
}

func (e *AlertEvaluator) evaluateRule(ev *ruleEvaluator) {
	rule := ev.rule

	// Get datasource
	ds, err := e.datasourceService.GetByID(rule.DatasourceID)
	if err != nil {
		return
	}

	// Execute query
	value, err := e.executeQuery(ds, rule.Query, rule.ForDuration)
	if err != nil {
		// Update rule status to error
		_ = e.db.Model(rule).Update("status", "error").Error
		return
	}

	// Update rule status to active
	_ = e.db.Model(rule).Updates(map[string]interface{}{
		"status":          "active",
		"last_evaluation": time.Now(),
	}).Error

	// PromQL semantics: a non-zero instant-vector value means the condition holds.
	shouldFire := value > 0

	if shouldFire {
		if !ev.state.firing {
			// Start firing state
			ev.state.firing = true
			ev.state.firingSince = time.Now()
			ev.state.consecutiveOK = 0
			ev.state.notified = false
		}

		// Check if fire duration threshold met
		firingDuration := time.Since(ev.state.firingSince)
		if firingDuration >= time.Duration(rule.ForDuration)*time.Second && !ev.state.notified {
			ev.state.notified = true
			_ = e.alertRuleService.RecordEvent(rule.ID, value, "firing")
			e.sendNotifications(rule, value, "firing")
		}
	} else {
		if ev.state.firing {
			// Alert resolved
			_ = e.alertRuleService.RecordEvent(rule.ID, 0, "resolved")
			e.sendNotifications(rule, 0, "resolved")
			ev.state.firing = false
			ev.state.consecutiveOK = 0
			ev.state.notified = false
		}
	}

	ev.state.lastValue = value
}

// sendNotifications delivers an alert event to every channel bound to the rule.
// Delivery is best-effort: a failing channel must not break the evaluator loop.
func (e *AlertEvaluator) sendNotifications(rule *AlertRule, value float64, status string) {
	if e.notificationService == nil {
		return
	}
	var channelIDs []uint64
	_ = json.Unmarshal(rule.NotificationChannels, &channelIDs)
	message := map[string]interface{}{
		"rule":     rule.Name,
		"severity": rule.Severity,
		"status":   status,
		"value":    value,
		"query":    rule.Query,
	}
	for _, channelID := range channelIDs {
		if err := e.notificationService.SendNotification(channelID, 0, rule.ID, rule.Name, rule.Severity, message); err != nil {
			// Best-effort: record the failure without stopping the loop.
			_ = e.db.Model(rule).Update("status", "error").Error
		}
	}
}

func (e *AlertEvaluator) executeQuery(ds *DatasourceResponse, query string, lookback int) (float64, error) {
	// Get datasource record for credentials
	var dsRecord PrometheusDatasource
	if err := e.db.First(&dsRecord, ds.ID).Error; err != nil {
		return 0, err
	}

	// Decrypt credentials
	password, err := secrets.Decrypt(dsRecord.Password)
	if err != nil {
		return 0, err
	}
	bearerToken, err := secrets.Decrypt(dsRecord.BearerToken)
	if err != nil {
		return 0, err
	}

	// Build query URL with proper encoding
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", ds.URL, url.QueryEscape(query))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ds.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return 0, err
	}

	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	} else if dsRecord.Username != "" {
		req.SetBasicAuth(dsRecord.Username, password)
	}

	client := &http.Client{Timeout: time.Duration(ds.Timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse Prometheus response
	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Status != "success" {
		return 0, fmt.Errorf("query failed: %s", result.Status)
	}

	// Extract value from result
	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	// Get first result value
	if len(result.Data.Result[0].Value) < 2 {
		return 0, nil
	}

	valueStr, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, nil
	}

	var value float64
	_, err = fmt.Sscanf(valueStr, "%f", &value)
	if err != nil {
		return 0, err
	}

	return value, nil
}
