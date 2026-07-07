// Package telemetry 提供 OpenTelemetry 分布式追踪功能
// 支持 Jaeger, Zipkin, Tempo 等追踪后端
//
// 基本用法:
//
//	tp, err := telemetry.InitTracer("my-service", "localhost:4318")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tp.Shutdown(context.Background())
package telemetry

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer 初始化 OpenTelemetry TracerProvider
// 创建 OTLP HTTP exporter 并配置全局 tracer
//
// 参数:
//
//	serviceName: 服务名称，用于标识追踪来源
//	otlpEndpoint: OTLP HTTP 端点，如 "localhost:4318"
//
// 返回:
//
//	*trace.TracerProvider: tracer provider 实例，需在程序退出时调用 Shutdown
//	error: 初始化失败时返回错误
func InitTracer(serviceName, otlpEndpoint string) (*trace.TracerProvider, error) {
	ctx := context.Background()

	// 如果没有配置 OTLP 端点，返回 no-op provider
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if otlpEndpoint == "" {
			// 返回 no-op provider（不采集追踪数据）
			return trace.NewTracerProvider(), nil
		}
	}

	// 创建 OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(), // 开发环境不使用 TLS
	)
	if err != nil {
		return nil, err
	}

	// 创建资源
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.8.3"),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建 TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithBatchTimeout(5*time.Second),
			trace.WithMaxExportBatchSize(512),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // 开发环境全采样，生产环境可改为 TraceIDRatioBased
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	return tp, nil
}
