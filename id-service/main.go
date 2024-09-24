//
// @package Showcase-SigNoz-Golang
//
// @file Todo main
// @copyright 2024-present Christoph Kappel <christoph@unexist.dev>
// @version $Id$
//
// This program can be distributed under the terms of the Apache License v2.0.
// See the file LICENSE for details.
//

package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/encoding/gzip"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/unexist/showcase-microservices-golang/adapter"
	"github.com/unexist/showcase-microservices-golang/infrastructure/middlewares"

	"log"
	"os"
)

func getEnvOrDefault(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}

	return value
}

func initTracer(ctx context.Context) *sdktrace.TracerProvider {
	var exporter sdktrace.SpanExporter
	var err error

	/* Create trace exporter */
	exporter, err = otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(getEnvOrDefault("APP_SIGNOZ_HOST_PORT", "localhost:4317")),
		otlptracegrpc.WithCompressor(gzip.Name),
	)

	if nil != err {
		log.Fatal(err)
	}

	/* Create processor */
	batcher := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(1000),
		sdktrace.WithMaxExportBatchSize(1000))

	/* Create resource */
	resource, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String("service.name", "id-service"),
			attribute.String("service.version", "1.0.0"),
		))
	if nil != err {
		log.Fatal(err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batcher),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	return provider
}

func main() {
	/* Init tracer */
	ctx := context.Background()

	provider := initTracer(ctx)
	defer func() {
		if err := provider.Shutdown(context.Background()); nil != err {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	idResource := adapter.NewIdResource()

	/* Finally start Gin */
	engine := gin.New()

	engine.Use(gin.Recovery())

	/* Create monitor */
	monitor := ginmetrics.GetMonitor()

	monitor.SetMetricPath("/metrics")
	monitor.SetSlowTime(10)
	monitor.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
	monitor.Use(engine)

	engine.Use(middlewares.HttpStatusMiddleware())
	engine.Use(middlewares.CorrelationMiddleware())
	engine.Use(middlewares.DefaultStructuredLogger())
	engine.Use(otelgin.Middleware("id-service"))

	idResource.RegisterRoutes(engine)

	log.Fatal(http.ListenAndServe(
		getEnvOrDefault("APP_ID_LISTEN_HOST_PORT", "localhost:8081"), engine))
}
