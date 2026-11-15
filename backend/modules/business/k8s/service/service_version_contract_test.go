package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestServiceVersionContract(t *testing.T) {
	t.Run("update requires version", func(t *testing.T) {
		r := gin.New()
		NewServiceHandler(NewServiceService(nil, nil), nil).RegisterRoutes(r)
		req := httptest.NewRequest(http.MethodPut, "/clusters/1/services/app?namespace=default", bytes.NewBufferString(`{"selector":{"app":"new"}}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var payload struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &payload)
		if payload.Code != 400 {
			t.Fatalf("unversioned update status=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("delete requires version", func(t *testing.T) {
		r := gin.New()
		NewServiceHandler(NewServiceService(nil, nil), nil).RegisterRoutes(r)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/clusters/1/services/app?namespace=default", nil))
		var payload struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &payload)
		if payload.Code != 400 {
			t.Fatalf("unversioned delete status=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("projection retains provider identity and named port", func(t *testing.T) {
		svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", UID: types.UID("provider-uid"), ResourceVersion: "43"}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
		svc.Spec.Ports[0].TargetPort.Type = 1
		svc.Spec.Ports[0].TargetPort.StrVal = "http"
		raw, err := json.Marshal(toItem(svc))
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(raw, &response); err != nil {
			t.Fatal(err)
		}
		if response["resourceVersion"] != "43" || response["uid"] != "provider-uid" {
			t.Fatalf("provider identity omitted: %s", raw)
		}
		if !bytes.Contains(raw, []byte(`"targetPortName":"http"`)) {
			t.Fatalf("named target port omitted: %s", raw)
		}
	})
}
