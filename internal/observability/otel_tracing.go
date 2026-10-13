package observability

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Metrics struct {
	RequestsTotal   metric.Int64Counter
	RequestDuration metric.Float64Histogram
	ErrorsTotal     metric.Int64Counter
}

func Initialize(ctx context.Context) (*sdkmetric.MeterProvider, *Metrics, *sdktrace.TracerProvider, error) {
	meterProvider, metrics, err := InitializeMeterProvider(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	tracerProvider, err := InitializeTracerProvider(ctx)
	if err != nil {
		if shutdownErr := meterProvider.Shutdown(ctx); shutdownErr != nil {
			return nil, nil, nil, shutdownErr
		}
		return nil, nil, nil, err
	}

	return meterProvider, metrics, tracerProvider, nil
}

func InitializeMeterProvider(ctx context.Context) (*sdkmetric.MeterProvider, *Metrics, error) {
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint("otel-collector:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName("web-scraper"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(15*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)

	meter := provider.Meter("web-analyzer")

	requestsTotal, err := meter.Int64Counter(
		"web_scraper_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"web_scraper_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, err
	}

	errorsTotal, err := meter.Int64Counter(
		"web_scraper_errors_total",
		metric.WithDescription("Total number of failed requests"),
	)
	if err != nil {
		return nil, nil, err
	}

	heapAllocBytes, err := meter.Int64ObservableGauge(
		"web_scraper_memory_heap_alloc_bytes",
		metric.WithDescription("Bytes of allocated heap objects"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, nil, err
	}

	heapInUseBytes, err := meter.Int64ObservableGauge(
		"web_scraper_memory_heap_in_use_bytes",
		metric.WithDescription("Bytes in in-use heap spans"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, nil, err
	}

	goroutines, err := meter.Int64ObservableGauge(
		"web_scraper_runtime_goroutines",
		metric.WithDescription("Number of active goroutines"),
		metric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return nil, nil, err
	}

	_, err = meter.RegisterCallback(
		func(_ context.Context, observer metric.Observer) error {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			observer.ObserveInt64(heapAllocBytes, int64(stats.HeapAlloc))
			observer.ObserveInt64(heapInUseBytes, int64(stats.HeapInuse))
			observer.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))
			return nil
		},
		heapAllocBytes,
		heapInUseBytes,

		goroutines,
	)
	if err != nil {
		return nil, nil, err
	}

	return provider, &Metrics{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
		ErrorsTotal:     errorsTotal,
	}, nil
}

func InitializeTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName("web-scraper"),
		),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}
