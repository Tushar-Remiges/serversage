package serversage

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitServersage(otelEndpoint string, serviceName string) (*sdktrace.TracerProvider, *sdkmetric.MeterProvider, error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn, err := initConn(otelEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize gRPC connection: %w", err)
	}

	svcName := semconv.ServiceNameKey.String(serviceName)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// The service name used to display traces in backends
			svcName,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tracerProvider, err := initTracerProvider(ctx, res, conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	meterProvider, err := initMeterProvider(ctx, res, conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	return tracerProvider, meterProvider, nil

}

// Initialize a gRPC connection to be used by both the tracer and meter
// providers.
func initConn(otelEndpoint string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(otelEndpoint,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	return conn, err
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func initTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdktrace.TracerProvider, error) {
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider, nil
}

// Initializes an OTLP exporter, and configures the corresponding meter provider.
func initMeterProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdkmetric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}

var (
	meter = otel.Meter("example.com/metrics")

	system_uptime_seconds, _ = meter.Float64Gauge("system_uptime_seconds", metric.WithDescription("The total system uptime in seconds."))

	http_requests_total, _ = meter.Int64Counter("http_requests_total", metric.WithDescription("The total number of HTTP requests."))

	http_request_duration_seconds, _ = meter.Float64Histogram("http_request_duration_seconds", metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1, 5, 10), metric.WithDescription("The duration of HTTP requests in seconds."))

	active_sessions, _ = meter.Float64Gauge("active_sessions", metric.WithDescription("The current number of active sessions."))
)

func RecordSystemUptimeSeconds(value float64) {

	system_uptime_seconds.Record(context.Background(), value)

}

func RecordHttpRequestsTotal(method string, status string, value int64) {

	http_requests_total.Add(context.Background(), value, metric.WithAttributes(attribute.String("method", method), attribute.String("status", status)))

}

func RecordHttpRequestDurationSeconds(method string, status string, value float64) {

	http_request_duration_seconds.Record(context.Background(), value, metric.WithAttributes(attribute.String("method", method), attribute.String("status", status)))

}

func RecordActiveSessions(user_type string, value float64) {

	active_sessions.Record(context.Background(), value, metric.WithAttributes(attribute.String("user_type", user_type)))

}
