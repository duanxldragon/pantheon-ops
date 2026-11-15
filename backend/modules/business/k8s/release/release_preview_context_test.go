package release

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadContextProjectsMeasuredUnchangedResourcesAndHidesConfig(t *testing.T) {
	replicas := int32(3)
	pod := corev1.PodSpec{Containers: []corev1.Container{{Name: "sidecar"}, {
		Name: "app", Env: []corev1.EnvVar{{Name: "SECRET_SENTINEL", Value: "private-value"}},
		EnvFrom:   []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-name"}}}},
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("128Mi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("512Mi")}},
	}}, Volumes: []corev1.Volume{{Name: "secret-volume"}}}
	for _, item := range []metav1.Object{
		&appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: pod}}},
		&appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: pod}}},
		&appsv1.DaemonSet{Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: pod}}},
	} {
		changes := previewWorkloadContext(item, "app")
		fields := map[string]string{}
		for _, change := range changes {
			require.Equal(t, "unchanged", change.Kind)
			require.Equal(t, change.Before, change.After)
			fields[change.Field] = change.Before
		}
		require.Equal(t, "250m", fields["cpuRequest"])
		require.Equal(t, "128Mi", fields["memoryRequest"])
		require.Equal(t, "1", fields["cpuLimit"])
		require.Equal(t, "512Mi", fields["memoryLimit"])
		require.Equal(t, "1", fields["environmentEntries"])
		require.Equal(t, "1", fields["configurationSources"])
		public, err := json.Marshal(changes)
		require.NoError(t, err)
		for _, secret := range []string{"SECRET_SENTINEL", "private-value", "secret-name", "secret-volume"} {
			require.NotContains(t, string(public), secret)
		}
	}
}

func TestReviewedReleaseHTTPRejectsUnsupportedMutationFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/releases", "/releases/preview", "/releases/1/rollback", "/releases/1/rollback/preview"} {
		for _, field := range []string{`"manifest":"hidden"`, `"config":{"password":"hidden"}`, `"resources":{"cpu":"2"}`} {
			router := gin.New()
			NewReleaseHandler(nil).RegisterRoutes(router)
			body := `{"name":"api","clusterId":1,"namespace":"default","workloadType":"deployment","workloadName":"api","image":"api:v2",` + field + `}`
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			var envelope struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			require.Equal(t, 400, envelope.Code, "unsupported intent must not reach service")
			require.NotContains(t, response.Body.String(), "hidden")
		}
	}
}

func TestReviewedReleaseHTTPRejectsTrailingAndOversizeJSON(t *testing.T) {
	for _, body := range []string{`{"idempotencyKey":"key"}{"resources":{}}`, `{"idempotencyKey":"` + strings.Repeat("x", 65536) + `"}`} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var value RollbackReleaseRequest
		require.Error(t, bindReviewedRelease(c, &value))
	}
}
