package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// Release review tokens have a separate MAC purpose from stored kubeconfigs.
// The existing root key works across replicas; rotation invalidates old reviews.
func releaseReviewMAC(payload []byte) ([]byte, error) {
	root, err := loadKey()
	if err != nil {
		return nil, err
	}
	derive := hmac.New(sha256.New, root)
	_, _ = derive.Write([]byte("pantheon-ops/k8s-release-review/v1"))
	mac := hmac.New(sha256.New, derive.Sum(nil))
	_, _ = mac.Write(payload)
	return mac.Sum(nil), nil
}

// SignReleaseReview signs a bounded provider-owned review payload.
func SignReleaseReview(payload []byte) (string, error) {
	if len(payload) > 2048 {
		return "", errors.New("k8s.release.preview_conflict")
	}
	mac, err := releaseReviewMAC(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

// OpenReleaseReview authenticates a review. Semantic/expiry checks belong to the provider.
func OpenReleaseReview(token string) ([]byte, error) {
	fail := errors.New("k8s.release.preview_conflict")
	if len(token) > 4096 {
		return nil, fail
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fail
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(payload) > 2048 {
		return nil, fail
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fail
	}
	expected, err := releaseReviewMAC(payload)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(expected, signature) {
		return nil, fail
	}
	return payload, nil
}
