package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestInitTracer(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		otlpEndpoint string
		wantErr      bool
	}{
		{
			name:         "with valid endpoint",
			serviceName:  "test-service",
			otlpEndpoint: "localhost:4318",
			wantErr:      false,
		},
		{
			name:         "with empty endpoint returns no-op provider",
			serviceName:  "test-service",
			otlpEndpoint: "",
			wantErr:      false,
		},
		{
			name:         "with empty service name",
			serviceName:  "",
			otlpEndpoint: "localhost:4318",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := InitTracer(tt.serviceName, tt.otlpEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitTracer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tp == nil {
				t.Error("InitTracer() returned nil TracerProvider")
				return
			}

			// 测试 Shutdown
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				t.Errorf("TracerProvider.Shutdown() error = %v", err)
			}
		})
	}
}

func TestInitTracerNoOp(t *testing.T) {
	// 测试无 OTLP 端点时返回 no-op provider
	tp, err := InitTracer("test", "")
	if err != nil {
		t.Errorf("InitTracer() unexpected error = %v", err)
	}
	if tp == nil {
		t.Error("InitTracer() returned nil TracerProvider")
	}

	// No-op provider 应该可以正常 Shutdown
	ctx := context.Background()
	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("TracerProvider.Shutdown() error = %v", err)
	}
}
