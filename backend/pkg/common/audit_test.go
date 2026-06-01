package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuditContextSetters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	SetAuditMetadata(ctx, "audit.title", BusinessUpdate)
	SetAuditParam(ctx, `{"id":1}`)
	SetAuditResult(ctx, `{"ok":true}`)
	SetAuditStatus(ctx, 2)
	SetAuditErrorMsg(ctx, "audit.error")

	if value, ok := ctx.Get(OperationLogTitleKey); !ok || value != "audit.title" {
		t.Fatalf("expected %s to be set", OperationLogTitleKey)
	}
	if value, ok := ctx.Get(OperationLogBusinessTypeKey); !ok || value != BusinessUpdate {
		t.Fatalf("expected %s to be set", OperationLogBusinessTypeKey)
	}
	if value, ok := ctx.Get(OperationLogParamKey); !ok || value != `{"id":1}` {
		t.Fatalf("expected %s to be set", OperationLogParamKey)
	}
	if value, ok := ctx.Get(OperationLogResultKey); !ok || value != `{"ok":true}` {
		t.Fatalf("expected %s to be set", OperationLogResultKey)
	}
	if value, ok := ctx.Get(OperationLogStatusKey); !ok || value != 2 {
		t.Fatalf("expected %s to be set", OperationLogStatusKey)
	}
	if value, ok := ctx.Get(OperationLogErrorMsgKey); !ok || value != "audit.error" {
		t.Fatalf("expected %s to be set", OperationLogErrorMsgKey)
	}
}
