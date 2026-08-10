package login

import "time"

// SystemLogLogin 登录日志模型
type SystemLogLogin struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	RequestID     string `gorm:"size:64;index"`
	Username      string `gorm:"size:64"`
	Ipaddr        string `gorm:"size:128"`
	LoginLocation string `gorm:"size:255"`
	Browser       string `gorm:"size:128"`
	Os            string `gorm:"size:128"`
	// No gorm default tag: LoginStatusFailure is 0, and GORM replaces zero-value
	// fields that carry a default tag, which silently records failures as success.
	Status    int
	Msg       string    `gorm:"size:255"`
	LoginTime time.Time `gorm:"default:null"`
}

func (SystemLogLogin) TableName() string {
	return "system_log_login"
}
