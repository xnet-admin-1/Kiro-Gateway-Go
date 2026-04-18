package client

import (
	"context"
	"io"
	"time"
)

// StalledStreamProtection protects against stalled streams
type StalledStreamProtection struct {
	gracePeriod time.Duration
	minSpeed    int64 // bytes per second
}

// NewStalledStreamProtection creates a new stalled stream protection
func NewStalledStreamProtection(gracePeriod time.Duration, minSpeed int64) *StalledStreamProtection {
	return &StalledStreamProtection{
		gracePeriod: gracePeriod,
		minSpeed:    minSpeed,
	}
}

// WrapReader wraps a reader with stalled stream protection
func (p *StalledStreamProtection) WrapReader(ctx context.Context, r io.Reader) io.Reader {
	return &stalledReader{
		reader:      r,
		ctx:         ctx,
		gracePeriod: p.gracePeriod,
		minSpeed:    p.minSpeed,
		lastRead:    time.Now(),
		totalBytes:  0,
		startTime:   time.Now(),
	}
}

type stalledReader struct {
	reader      io.Reader
	ctx         context.Context
	gracePeriod time.Duration
	minSpeed    int64
	lastRead    time.Time
	totalBytes  int64
	startTime   time.Time
}

func (r *stalledReader) Read(p []byte) (int, error) {
	// Check if context is cancelled
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	// Check if stream is stalled (no data within grace period)
	if time.Since(r.lastRead) > r.gracePeriod {
		return 0, context.DeadlineExceeded
	}

	// Check if stream is too slow
	elapsed := time.Since(r.startTime)
	if elapsed > time.Second && r.totalBytes > 0 {
		currentSpeed := r.totalBytes / int64(elapsed.Seconds())
		if currentSpeed < r.minSpeed {
			return 0, context.DeadlineExceeded
		}
	}

	n, err := r.reader.Read(p)
	if n > 0 {
		r.lastRead = time.Now()
		r.totalBytes += int64(n)
	}

	return n, err
}
