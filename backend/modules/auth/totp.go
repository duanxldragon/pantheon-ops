package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpIssuer       = "Pantheon Base"
	totpPeriod       = int64(30)
	totpDigits       = 6
	totpAllowedSkew  = 1
	totpSecretLength = 20
)

func generateTOTPSecret() (string, error) {
	raw := make([]byte, totpSecretLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

func buildTOTPURL(username, secret string) string {
	account := strings.TrimSpace(username)
	if account == "" {
		account = "pantheon-user"
	}
	label := url.PathEscape(totpIssuer + ":" + account)
	values := url.Values{}
	values.Set("secret", strings.TrimSpace(secret))
	values.Set("issuer", totpIssuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", strconv.Itoa(totpDigits))
	values.Set("period", strconv.FormatInt(totpPeriod, 10))
	return "otpauth://totp/" + label + "?" + values.Encode()
}

func validateTOTPCode(secret string, code string, now time.Time) bool {
	normalizedCode := strings.TrimSpace(code)
	if len(normalizedCode) != totpDigits {
		return false
	}
	for _, ch := range normalizedCode {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	counter := now.Unix() / totpPeriod
	for offset := -totpAllowedSkew; offset <= totpAllowedSkew; offset++ {
		if generateTOTPCode(secret, counter+int64(offset)) == normalizedCode {
			return true
		}
	}
	return false
}

func generateTOTPCode(secret string, counter int64) string {
	key, err := decodeBase32NoPadding(secret)
	if err != nil {
		return ""
	}
	var payload [8]byte
	binary.BigEndian.PutUint64(payload[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(payload[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	modulo := int(math.Pow10(totpDigits))
	return fmt.Sprintf("%0*d", totpDigits, binaryCode%modulo)
}

func decodeBase32NoPadding(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	if normalized == "" {
		return nil, fmt.Errorf("auth.mfa.secret.required")
	}
	padding := len(normalized) % 8
	if padding != 0 {
		normalized += strings.Repeat("=", 8-padding)
	}
	return base32.StdEncoding.DecodeString(normalized)
}
