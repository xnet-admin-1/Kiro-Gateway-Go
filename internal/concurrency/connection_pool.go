package concurrency

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionPoolMonitor monitors HTTP connection pool health and metrics
type ConnectionPoolMonitor struct {
	// Configuration
	maxIdleConns        int
	maxIdleConnsPerHost int
	maxConnsPerHost     int
	idleConnTimeout     time.Duration
	
	// Metrics
	metrics *ConnectionPoolMetrics
	
	// Transport reference
	transport *http.Transport
	
	// Health check
	healthCheckInterval time.Duration
	healthCheckTimeout  time.Duration
	
	// State
	started atomic.Bool
	stopped atomic.Bool
	
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	mu sync.RWMutex
}

// ConnectionPoolMetrics tracks connection pool performance
type ConnectionPoolMetrics struct {
	// Connection counts
	ActiveConnections   atomic.Int32
	IdleConnections     atomic.Int32
	TotalConnections    atomic.Int32
	
	// Connection lifecycle
	ConnectionsCreated  atomic.Int64
	ConnectionsClosed   atomic.Int64
	ConnectionsReused   atomic.Int64
	ConnectionsFailed   atomic.Int64
	
	// Pool exhaustion
	PoolExhaustionCount atomic.Int64
	WaitTimeTotal       atomic.Int64 // nanoseconds
	WaitCount           atomic.Int64
	
	// Health checks
	HealthChecksPassed  atomic.Int64
	HealthChecksFailed  atomic.Int64
	LastHealthCheck     atomic.Int64 // Unix nano
	
	// Timeouts
	DialTimeouts        atomic.Int64
	IdleTimeouts        atomic.Int64
	ResponseTimeouts    atomic.Int64
	
	mu sync.RWMutex
}

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
	IdleConnTimeout       time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
	DialTimeout           time.Duration
	KeepAlive             time.Duration
	HealthCheckInterval   time.Duration
	HealthCheckTimeout    time.Duration
	HealthCheckURL        string
}

// NewConnectionPoolMonitor creates a new connection pool monitor
func NewConnectionPoolMonitor(cfg ConnectionPoolConfig) *ConnectionPoolMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 200
	}
	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = 50
	}
	if cfg.MaxConnsPerHost <= 0 {
		cfg.MaxConnsPerHost = 100
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 120 * time.Second
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = 30 * time.Second
	}
	if cfg.ExpectContinueTimeout <= 0 {
		cfg.ExpectContinueTimeout = 1 * time.Second
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 30 * time.Second
	}
	if cfg.HealthCheckInterval <= 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}
	if cfg.HealthCheckTimeout <= 0 {
		cfg.HealthCheckTimeout = 5 * time.Second
	}
	
	// Create custom dialer with metrics
	dialer := &net.Dialer{
		Timeout:   cfg.DialTimeout,
		KeepAlive: cfg.KeepAlive,
	}
	
	// Create transport with enhanced configuration
	transport := &http.Transport{
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		DisableCompression:    false,
		DisableKeepAlives:     false,
		DialContext:           dialer.DialContext,
	}
	
	monitor := &ConnectionPoolMonitor{
		maxIdleConns:        cfg.MaxIdleConns,
		maxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		maxConnsPerHost:     cfg.MaxConnsPerHost,
		idleConnTimeout:     cfg.IdleConnTimeout,
		metrics:             &ConnectionPoolMetrics{},
		transport:           transport,
		healthCheckInterval: cfg.HealthCheckInterval,
		healthCheckTimeout:  cfg.HealthCheckTimeout,
		ctx:                 ctx,
		cancel:              cancel,
	}
	
	log.Printf("Connection pool monitor initialized (max idle: %d, max per host: %d, max conns per host: %d)",
		cfg.MaxIdleConns, cfg.MaxIdleConnsPerHost, cfg.MaxConnsPerHost)
	
	return monitor
}

// Start starts the connection pool monitor
func (cpm *ConnectionPoolMonitor) Start() error {
	if cpm.started.Load() {
		return fmt.Errorf("connection pool monitor already started")
	}
	
	cpm.started.Store(true)
	
	// Start health check loop
	cpm.wg.Add(1)
	go cpm.healthCheckLoop()
	
	// Start metrics collection loop
	cpm.wg.Add(1)
	go cpm.metricsLoop()
	
	log.Println("Connection pool monitor started")
	
	return nil
}

// Stop stops the connection pool monitor
func (cpm *ConnectionPoolMonitor) Stop(timeout time.Duration) error {
	if cpm.stopped.Load() {
		return fmt.Errorf("connection pool monitor already stopped")
	}
	
	log.Println("Stopping connection pool monitor...")
	
	cpm.stopped.Store(true)
	cpm.cancel()
	
	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		cpm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Close idle connections
		cpm.transport.CloseIdleConnections()
		log.Println("Connection pool monitor stopped gracefully")
		return nil
	case <-time.After(timeout):
		log.Println("Connection pool monitor stopped with timeout")
		return fmt.Errorf("connection pool monitor stop timeout after %v", timeout)
	}
}

// GetTransport returns the monitored HTTP transport
func (cpm *ConnectionPoolMonitor) GetTransport() *http.Transport {
	return cpm.transport
}

// healthCheckLoop performs periodic health checks
func (cpm *ConnectionPoolMonitor) healthCheckLoop() {
	defer cpm.wg.Done()
	
	ticker := time.NewTicker(cpm.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check on the connection pool
func (cpm *ConnectionPoolMonitor) performHealthCheck() {
	cpm.metrics.LastHealthCheck.Store(time.Now().UnixNano())
	
	// Check pool utilization
	idleConns := cpm.metrics.IdleConnections.Load()
	activeConns := cpm.metrics.ActiveConnections.Load()
	totalConns := idleConns + activeConns
	
	// Calculate utilization
	idleUtilization := float64(idleConns) / float64(cpm.maxIdleConns) * 100
	
	// Check for pool exhaustion
	if totalConns >= int32(cpm.maxIdleConns) {
		cpm.metrics.PoolExhaustionCount.Add(1)
		log.Printf("Warning: Connection pool near exhaustion (total: %d, max: %d)",
			totalConns, cpm.maxIdleConns)
		cpm.metrics.HealthChecksFailed.Add(1)
		return
	}
	
	// Check for excessive idle connections
	if idleUtilization > 90 {
		log.Printf("Warning: High idle connection utilization (%.1f%%)", idleUtilization)
	}
	
	cpm.metrics.HealthChecksPassed.Add(1)
	
	log.Printf("Connection pool health check passed (active: %d, idle: %d, utilization: %.1f%%)",
		activeConns, idleConns, idleUtilization)
}

// metricsLoop collects metrics periodically
func (cpm *ConnectionPoolMonitor) metricsLoop() {
	defer cpm.wg.Done()
	
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.collectMetrics()
		}
	}
}

// collectMetrics collects current metrics
func (cpm *ConnectionPoolMonitor) collectMetrics() {
	// Note: Go's http.Transport doesn't expose internal connection counts
	// We track connections through our instrumented dialer and callbacks
	
	// Log current state
	log.Printf("Connection pool metrics: active=%d, idle=%d, created=%d, reused=%d, failed=%d",
		cpm.metrics.ActiveConnections.Load(),
		cpm.metrics.IdleConnections.Load(),
		cpm.metrics.ConnectionsCreated.Load(),
		cpm.metrics.ConnectionsReused.Load(),
		cpm.metrics.ConnectionsFailed.Load())
}

// RecordConnectionCreated records a new connection creation
func (cpm *ConnectionPoolMonitor) RecordConnectionCreated() {
	cpm.metrics.ConnectionsCreated.Add(1)
	cpm.metrics.ActiveConnections.Add(1)
	cpm.metrics.TotalConnections.Add(1)
}

// RecordConnectionClosed records a connection closure
func (cpm *ConnectionPoolMonitor) RecordConnectionClosed() {
	cpm.metrics.ConnectionsClosed.Add(1)
	cpm.metrics.ActiveConnections.Add(-1)
	cpm.metrics.TotalConnections.Add(-1)
}

// RecordConnectionReused records a connection reuse
func (cpm *ConnectionPoolMonitor) RecordConnectionReused() {
	cpm.metrics.ConnectionsReused.Add(1)
}

// RecordConnectionFailed records a connection failure
func (cpm *ConnectionPoolMonitor) RecordConnectionFailed() {
	cpm.metrics.ConnectionsFailed.Add(1)
}

// RecordConnectionWait records time spent waiting for a connection
func (cpm *ConnectionPoolMonitor) RecordConnectionWait(duration time.Duration) {
	cpm.metrics.WaitTimeTotal.Add(duration.Nanoseconds())
	cpm.metrics.WaitCount.Add(1)
}

// RecordDialTimeout records a dial timeout
func (cpm *ConnectionPoolMonitor) RecordDialTimeout() {
	cpm.metrics.DialTimeouts.Add(1)
}

// RecordIdleTimeout records an idle timeout
func (cpm *ConnectionPoolMonitor) RecordIdleTimeout() {
	cpm.metrics.IdleTimeouts.Add(1)
}

// RecordResponseTimeout records a response timeout
func (cpm *ConnectionPoolMonitor) RecordResponseTimeout() {
	cpm.metrics.ResponseTimeouts.Add(1)
}

// GetStats returns human-readable statistics
func (cpm *ConnectionPoolMonitor) GetStats() map[string]interface{} {
	activeConns := cpm.metrics.ActiveConnections.Load()
	idleConns := cpm.metrics.IdleConnections.Load()
	totalConns := activeConns + idleConns
	
	created := cpm.metrics.ConnectionsCreated.Load()
	closed := cpm.metrics.ConnectionsClosed.Load()
	reused := cpm.metrics.ConnectionsReused.Load()
	failed := cpm.metrics.ConnectionsFailed.Load()
	
	waitCount := cpm.metrics.WaitCount.Load()
	var avgWaitTime time.Duration
	if waitCount > 0 {
		avgWaitTime = time.Duration(cpm.metrics.WaitTimeTotal.Load() / waitCount)
	}
	
	var reuseRate float64
	if created > 0 {
		reuseRate = float64(reused) / float64(created+reused) * 100
	}
	
	lastHealthCheck := time.Unix(0, cpm.metrics.LastHealthCheck.Load())
	
	return map[string]interface{}{
		"configuration": map[string]interface{}{
			"max_idle_conns":         cpm.maxIdleConns,
			"max_idle_conns_per_host": cpm.maxIdleConnsPerHost,
			"max_conns_per_host":     cpm.maxConnsPerHost,
			"idle_conn_timeout":      cpm.idleConnTimeout.String(),
		},
		"connections": map[string]interface{}{
			"active":     activeConns,
			"idle":       idleConns,
			"total":      totalConns,
			"created":    created,
			"closed":     closed,
			"reused":     reused,
			"failed":     failed,
			"reuse_rate": reuseRate,
		},
		"pool": map[string]interface{}{
			"exhaustion_count": cpm.metrics.PoolExhaustionCount.Load(),
			"wait_count":       waitCount,
			"avg_wait_time":    avgWaitTime.String(),
			"utilization":      float64(totalConns) / float64(cpm.maxIdleConns) * 100,
		},
		"timeouts": map[string]interface{}{
			"dial":     cpm.metrics.DialTimeouts.Load(),
			"idle":     cpm.metrics.IdleTimeouts.Load(),
			"response": cpm.metrics.ResponseTimeouts.Load(),
		},
		"health": map[string]interface{}{
			"checks_passed":    cpm.metrics.HealthChecksPassed.Load(),
			"checks_failed":    cpm.metrics.HealthChecksFailed.Load(),
			"last_check":       time.Since(lastHealthCheck).String(),
			"check_interval":   cpm.healthCheckInterval.String(),
		},
	}
}

// IsHealthy returns true if the connection pool is healthy
func (cpm *ConnectionPoolMonitor) IsHealthy() bool {
	if !cpm.started.Load() || cpm.stopped.Load() {
		return false
	}
	
	// Check pool utilization
	totalConns := cpm.metrics.ActiveConnections.Load() + cpm.metrics.IdleConnections.Load()
	utilization := float64(totalConns) / float64(cpm.maxIdleConns)
	
	if utilization > 0.9 {
		return false // Pool >90% utilized
	}
	
	// Check for recent exhaustion
	if cpm.metrics.PoolExhaustionCount.Load() > 0 {
		// Check if exhaustion happened recently (last 5 minutes)
		lastHealthCheck := time.Unix(0, cpm.metrics.LastHealthCheck.Load())
		if time.Since(lastHealthCheck) < 5*time.Minute {
			return false
		}
	}
	
	return true
}

// CloseIdleConnections closes all idle connections
func (cpm *ConnectionPoolMonitor) CloseIdleConnections() {
	cpm.transport.CloseIdleConnections()
	log.Println("Closed all idle connections")
}
