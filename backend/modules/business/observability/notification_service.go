package observability

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"time"

	"pantheon-base/modules/business/secrets"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var notificationSecretFields = []string{"password", "secret", "webhookUrl"}

// NotificationService manages notification channels and sends alerts.
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// Migrate creates or updates database schema.
func (s *NotificationService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&NotificationChannel{}, &NotificationLog{}); err != nil {
		return err
	}
	return s.encryptStoredChannelConfigs()
}

// CreateChannel adds a new notification channel.
func (s *NotificationService) CreateChannel(dto NotificationChannelDTO, createdBy string, scopes ...*common.DataScopeReq) (*NotificationChannelResponse, error) {
	protectedConfig, err := secrets.EncryptConfigFields(dto.Config, notificationSecretFields...)
	if err != nil {
		return nil, err
	}
	configJSON, err := json.Marshal(protectedConfig)
	if err != nil {
		return nil, err
	}

	record := &NotificationChannel{
		DeptID:    ownedDeptID(firstDataScope(scopes)),
		Name:      dto.Name,
		Type:      dto.Type,
		Enabled:   dto.Enabled,
		Config:    datatypes.JSON(configJSON),
		Status:    "active",
		CreatedBy: createdBy,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return s.toChannelResponse(record), nil
}

// ListChannels returns all notification channels.
func (s *NotificationService) ListChannels(scopes ...*common.DataScopeReq) (*NotificationChannelListResponse, error) {
	var records []NotificationChannel
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]NotificationChannelResponse, len(records))
	for i, r := range records {
		items[i] = *s.toChannelResponse(&r)
	}

	return &NotificationChannelListResponse{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// GetChannelByID retrieves a channel by ID.
func (s *NotificationService) GetChannelByID(id uint64, scopes ...*common.DataScopeReq) (*NotificationChannelResponse, error) {
	var record NotificationChannel
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toChannelResponse(&record), nil
}

// UpdateChannel modifies an existing channel.
func (s *NotificationService) UpdateChannel(id uint64, dto NotificationChannelDTO, scopes ...*common.DataScopeReq) error {
	var record NotificationChannel
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
		return err
	}
	var storedConfig map[string]interface{}
	if err := json.Unmarshal(record.Config, &storedConfig); err != nil {
		return err
	}
	config := secrets.PreserveStoredFields(dto.Config, storedConfig, notificationSecretFields...)
	protectedConfig, err := secrets.EncryptConfigFields(config, notificationSecretFields...)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(protectedConfig)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"name":    dto.Name,
		"type":    dto.Type,
		"enabled": dto.Enabled,
		"config":  configJSON,
	}

	return s.db.Model(&NotificationChannel{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteChannel removes a channel.
func (s *NotificationService) DeleteChannel(id uint64, scopes ...*common.DataScopeReq) error {
	result := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).Where("id = ?", id).Delete(&NotificationChannel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SendNotification sends an alert notification through a channel.
func (s *NotificationService) SendNotification(channelID, eventID, ruleID uint64, ruleName, severity string, message map[string]interface{}) error {
	var channel NotificationChannel
	if err := s.db.First(&channel, channelID).Error; err != nil {
		return err
	}

	if !channel.Enabled {
		return fmt.Errorf("channel disabled")
	}

	var config map[string]interface{}
	if err := json.Unmarshal(channel.Config, &config); err != nil {
		return err
	}
	config, err := secrets.DecryptConfigFields(config, notificationSecretFields...)
	if err != nil {
		return err
	}

	switch channel.Type {
	case "email":
		err = s.sendEmail(config, ruleName, severity, message)
	case "webhook":
		err = s.sendWebhook(config, ruleName, severity, message)
	case "slack":
		err = s.sendSlack(config, ruleName, severity, message)
	case "dingtalk":
		err = s.sendDingTalk(config, ruleName, severity, message)
	case "feishu":
		err = s.sendFeishu(config, ruleName, severity, message)
	default:
		err = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	// Log notification
	status := "success"
	errorMsg := ""
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	log := &NotificationLog{
		DeptID:    channel.DeptID,
		ChannelID: channelID,
		EventID:   eventID,
		RuleID:    ruleID,
		RuleName:  ruleName,
		Severity:  severity,
		Status:    status,
		Error:     errorMsg,
		SentAt:    time.Now(),
	}

	_ = s.db.Create(log).Error

	return err
}

func (s *NotificationService) sendEmail(config map[string]interface{}, ruleName, severity string, message map[string]interface{}) error {
	recipients, _ := config["recipients"].([]interface{})
	smtpHost, _ := config["smtpHost"].(string)
	smtpPort, _ := config["smtpPort"].(float64)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	fromAddr, _ := config["fromAddr"].(string)

	if len(recipients) == 0 || smtpHost == "" {
		return fmt.Errorf("missing email configuration")
	}

	subject := fmt.Sprintf("[%s] Alert: %s", severity, ruleName)
	body := fmt.Sprintf("Alert Rule: %s\nSeverity: %s\n\n%v", ruleName, severity, message)

	auth := smtp.PlainAuth("", username, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, int(smtpPort))

	to := make([]string, len(recipients))
	for i, r := range recipients {
		to[i] = r.(string)
	}

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to[0], subject, body))

	return smtp.SendMail(addr, auth, fromAddr, to, msg)
}

func (s *NotificationService) sendWebhook(config map[string]interface{}, ruleName, severity string, message map[string]interface{}) error {
	url, _ := config["url"].(string)
	method, _ := config["method"].(string)
	timeout, _ := config["timeout"].(float64)

	if url == "" {
		return fmt.Errorf("missing webhook url")
	}
	if method == "" {
		method = "POST"
	}
	if timeout == 0 {
		timeout = 10
	}

	payload := map[string]interface{}{
		"ruleName":  ruleName,
		"severity":  severity,
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *NotificationService) sendSlack(config map[string]interface{}, ruleName, severity string, message map[string]interface{}) error {
	webhookURL, _ := config["webhookUrl"].(string)
	channel, _ := config["channel"].(string)
	username, _ := config["username"].(string)

	if webhookURL == "" {
		return fmt.Errorf("missing slack webhook url")
	}

	color := "warning"
	if severity == "critical" {
		color = "danger"
	} else if severity == "info" {
		color = "good"
	}

	payload := map[string]interface{}{
		"username": username,
		"channel":  channel,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     fmt.Sprintf("[%s] %s", severity, ruleName),
				"text":      fmt.Sprintf("%v", message),
				"timestamp": time.Now().Unix(),
			},
		},
	}

	return s.sendWebhook(map[string]interface{}{
		"url":    webhookURL,
		"method": "POST",
	}, ruleName, severity, payload)
}

func (s *NotificationService) sendDingTalk(config map[string]interface{}, ruleName, severity string, message map[string]interface{}) error {
	webhookURL, _ := config["webhookUrl"].(string)
	secret, _ := config["secret"].(string)

	if webhookURL == "" {
		return fmt.Errorf("missing dingtalk webhook url")
	}

	timestamp := time.Now().UnixMilli()
	sign := ""
	if secret != "" {
		sign = s.generateDingTalkSign(timestamp, secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
	}

	text := fmt.Sprintf("## [%s] %s\n\n%v", severity, ruleName, message)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": ruleName,
			"text":  text,
		},
	}

	return s.sendWebhook(map[string]interface{}{
		"url":    webhookURL,
		"method": "POST",
	}, ruleName, severity, payload)
}

func (s *NotificationService) sendFeishu(config map[string]interface{}, ruleName, severity string, message map[string]interface{}) error {
	webhookURL, _ := config["webhookUrl"].(string)
	secret, _ := config["secret"].(string)

	if webhookURL == "" {
		return fmt.Errorf("missing feishu webhook url")
	}

	timestamp := time.Now().Unix()
	sign := ""
	if secret != "" {
		sign = s.generateFeishuSign(timestamp, secret)
	}

	text := fmt.Sprintf("[%s] %s\n\n%v", severity, ruleName, message)

	payload := map[string]interface{}{
		"timestamp": timestamp,
		"sign":      sign,
		"msg_type":  "text",
		"content": map[string]string{
			"text": text,
		},
	}

	return s.sendWebhook(map[string]interface{}{
		"url":    webhookURL,
		"method": "POST",
	}, ruleName, severity, payload)
}

func (s *NotificationService) generateDingTalkSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *NotificationService) generateFeishuSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *NotificationService) toChannelResponse(r *NotificationChannel) *NotificationChannelResponse {
	var config map[string]interface{}
	_ = json.Unmarshal(r.Config, &config)
	config = secrets.RedactConfigFields(config, notificationSecretFields...)

	return &NotificationChannelResponse{
		ID:        r.ID,
		DeptID:    r.DeptID,
		Name:      r.Name,
		Type:      r.Type,
		Enabled:   r.Enabled,
		Config:    config,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		CreatedBy: r.CreatedBy,
	}
}

func (s *NotificationService) encryptStoredChannelConfigs() error {
	var records []NotificationChannel
	if err := s.db.Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		var config map[string]interface{}
		if err := json.Unmarshal(records[i].Config, &config); err != nil {
			return err
		}
		protected, err := secrets.EncryptConfigFields(config, notificationSecretFields...)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(protected)
		if err != nil {
			return err
		}
		if err := s.db.Model(&NotificationChannel{}).Where("id = ?", records[i].ID).Update("config", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}
