package login

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/gin-gonic/gin"
)

func TestRefreshRotationHasOneWinnerAcrossHandlers(t *testing.T) {
	db := setupHandlerTestDB(t)
	rdb := testredis.OpenDB(t, testredis.DBLogin)
	previous := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = previous })
	u := seedHandlerUser(t, db, "refresh_rotation_user", "pass123")
	sess := SystemUserSession{SessionID: "rotation-session", UserID: u.ID, RefreshJTI: "old-jti", RefreshExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&sess).Error; err != nil {
		t.Fatal(err)
	}
	old := authtoken.NewRefreshToken()
	if err := authtoken.StoreRefresh(context.Background(), rdb, old, u.ID, sess.SessionID, time.Hour); err != nil {
		t.Fatal(err)
	}
	// Independent runtimes simulate two server instances sharing DB and Redis.
	handlers := []*AuthHandler{NewAuthHandler(NewRuntime(db, testCredentialRepo(db))), NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))}
	gin.SetMode(gin.TestMode)
	const requests = 16
	start := make(chan struct{})
	type result struct {
		body    string
		refresh string
	}
	results := make(chan result, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/auth/refresh", strings.NewReader(`{"refreshToken":"`+old+`"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			handlers[i%len(handlers)].RefreshTokenHandler(c)
			item := result{body: w.Body.String()}
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == commonhttp.CookieRefreshToken {
					item.refresh = cookie.Value
				}
			}
			results <- item
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	var nextToken string
	for item := range results {
		body := item.body
		var response struct {
			Code int `json:"code"`
			Data struct {
				RefreshToken string `json:"refreshToken"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &response); err != nil {
			t.Fatalf("response=%s: %v", body, err)
		}
		if response.Code == common.CodeSuccess {
			successes++
			nextToken = item.refresh
		}
	}
	t.Logf("parallel requests=%d successful pairs=%d", requests, successes)
	if successes != 1 {
		t.Fatalf("same old refresh produced %d successful pairs; want one", successes)
	}
	if _, _, err := authtoken.ValidateRefresh(context.Background(), rdb, old); err == nil {
		t.Fatal("old token remains reusable")
	}
	if _, _, err := authtoken.ValidateRefresh(context.Background(), rdb, nextToken); err != nil {
		t.Fatalf("new refresh invalid: %v", err)
	}
}

func TestRefreshRotationFailureConsumesOldToken(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBLogin)
	previous := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = previous })
	token := authtoken.NewRefreshToken()
	if err := authtoken.StoreRefresh(context.Background(), rdb, token, 42, "failed-session", time.Hour); err != nil {
		t.Fatal(err)
	}
	// Database unavailability must never return credentials or restore a token
	// after consuming it; the client must log in again after an uncertain failure.
	h := NewAuthHandler(NewRuntime(nil, nil))
	c, w := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", strings.NewReader(`{"refreshToken":"`+token+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.RefreshTokenHandler(c)
	code, data := decodeHandlerResponse(t, w)
	if code == common.CodeSuccess || data["refreshToken"] != nil {
		t.Fatal("database failure returned credentials")
	}
	if _, _, err := authtoken.ValidateRefresh(context.Background(), rdb, token); err == nil {
		t.Fatal("failed refresh can replay the old token")
	}
}
