package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dbmetrics "github.com/remiges-tech/serversage/example/otelc/metrics"
	"github.com/remiges-tech/serversage/example/promc/metrics"
)

func main() {

	tracerProvider, meterProvider, err := dbmetrics.InitServersage("0.0.0.0:4317", "otelc")
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shutdown tracer provider: %v", err)
		}
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shutdown meter provider: %v", err)
		}
	}()
	r := gin.Default()

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Middleware to record request duration
	r.Use(requestDurationMiddleware())

	// Existing handler adapted for Gin
	r.GET("/", func(c *gin.Context) {
		// Increment the http_requests_total metric using the generated wrapper function.
		dbmetrics.RecordHttpRequestsTotal(c.Request.Method, http.StatusText(http.StatusOK), 1)

		// wait for random time between 1 us to 10 seconds
		time.Sleep(time.Duration(rand.Intn(10_000_000)) * time.Microsecond)

		// Respond with a simple message.
		c.String(http.StatusOK, "Hello, world!")
	})

	// Start system uptime monitoring in a separate goroutine
	go updateSystemUptime()

	// Start server
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	r.Run(":" + port)
}

func requestDurationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next() // Process request
		duration := time.Since(startTime).Seconds()

		// Log the observed duration for debugging
		log.Printf("Observed request duration: %f seconds\n", duration)

		// Observe request duration
		metrics.RecordHttpRequestDurationSeconds(metrics.Method(c.Request.Method),
			metrics.Status(http.StatusText(c.Writer.Status())),
			duration)
	}
}

func updateSystemUptime() {
	startTime := time.Now()
	for {
		uptime := time.Since(startTime).Seconds()
		metrics.RecordSystemUptimeSeconds(uptime)
		time.Sleep(5 * time.Second)
	}
}
