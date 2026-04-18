package client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNewStalledStreamProtection(t *testing.T) {
	gracePeriod := 5 * time.Minute
	minSpeed := int64(1024)

	protection := NewStalledStreamProtection(gracePeriod, minSpeed)

	if protection.gracePeriod != gracePeriod {
		t.Errorf("Expected gracePeriod %v, got %v", gracePeriod, protection.gracePeriod)
	}
	if protection.minSpeed != minSpeed {
		t.Errorf("Expected minSpeed %d, got %d", minSpeed, protection.minSpeed)
	}
}

func TestStalledStreamProtection_WrapReader(t *testing.T) {
	protection := NewStalledStreamProtection(5*time.Minute, 1024)
	reader := strings.NewReader("test data")
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)

	if wrapped == nil {
		t.Error("Expected wrapped reader, got nil")
	}

	// Verify it's a stalledReader
	stalledReader, ok := wrapped.(*stalledReader)
	if !ok {
		t.Error("Expected stalledReader type")
	}

	if stalledReader.reader != reader {
		t.Error("Expected wrapped reader to contain original reader")
	}
	if stalledReader.ctx != ctx {
		t.Error("Expected wrapped reader to contain context")
	}
}

func TestStalledReader_Read_Normal(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		bufSize  int
		expected string
	}{
		{
			name:     "small buffer",
			data:     "hello world",
			bufSize:  5,
			expected: "hello",
		},
		{
			name:     "large buffer",
			data:     "test",
			bufSize:  10,
			expected: "test",
		},
		{
			name:     "empty data",
			data:     "",
			bufSize:  10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protection := NewStalledStreamProtection(5*time.Minute, 0)
			reader := strings.NewReader(tt.data)
			ctx := context.Background()

			wrapped := protection.WrapReader(ctx, reader)
			buf := make([]byte, tt.bufSize)

			n, err := wrapped.Read(buf)

			if tt.data == "" {
				if err != io.EOF {
					t.Errorf("Expected EOF for empty data, got %v", err)
				}
				return
			}

			if err != nil && err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}

			result := string(buf[:n])
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestStalledReader_Read_ContextCancelled(t *testing.T) {
	protection := NewStalledStreamProtection(5*time.Minute, 1024)
	reader := strings.NewReader("test data")
	ctx, cancel := context.WithCancel(context.Background())

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 10)

	// Cancel context before reading
	cancel()

	_, err := wrapped.Read(buf)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestStalledReader_Read_StalledStream(t *testing.T) {
	// Use very short grace period for testing
	protection := NewStalledStreamProtection(10*time.Millisecond, 1024)
	reader := &slowReader{data: "test data", delay: 20 * time.Millisecond}
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 10)

	// First read should succeed
	n, err := wrapped.Read(buf)
	if err != nil {
		t.Errorf("First read failed: %v", err)
	}
	if n == 0 {
		t.Error("Expected to read some data")
	}

	// Wait for grace period to expire
	time.Sleep(15 * time.Millisecond)

	// Second read should fail due to stalled stream
	_, err = wrapped.Read(buf)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded for stalled stream, got %v", err)
	}
}

func TestStalledReader_Read_SlowStream(t *testing.T) {
	// Set minimum speed requirement
	protection := NewStalledStreamProtection(5*time.Minute, 1000) // 1000 bytes/sec
	reader := &slowReader{data: strings.Repeat("x", 100), delay: 200 * time.Millisecond}
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 10)

	// Read some data to establish baseline
	for i := 0; i < 5; i++ {
		_, err := wrapped.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Read %d failed: %v", i, err)
		}
	}

	// Wait for speed calculation to kick in
	time.Sleep(1100 * time.Millisecond)

	// Next read should fail due to slow speed
	_, err := wrapped.Read(buf)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded for slow stream, got %v", err)
	}
}

func TestStalledReader_Read_FastStream(t *testing.T) {
	protection := NewStalledStreamProtection(5*time.Minute, 100) // 100 bytes/sec
	reader := strings.NewReader(strings.Repeat("x", 1000))
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 100)

	// Read data quickly - should not trigger slow stream detection
	totalRead := 0
	for totalRead < 500 {
		n, err := wrapped.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Fast stream read failed: %v", err)
		}
		totalRead += n
		if err == io.EOF {
			break
		}
	}

	if totalRead < 500 {
		t.Errorf("Expected to read at least 500 bytes, got %d", totalRead)
	}
}

// slowReader simulates a slow reader for testing
type slowReader struct {
	data  string
	pos   int
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	// Simulate slow reading
	time.Sleep(r.delay)

	// Read one byte at a time to simulate slow stream
	n := 1
	if len(p) < n {
		n = len(p)
	}
	if r.pos+n > len(r.data) {
		n = len(r.data) - r.pos
	}

	copy(p, r.data[r.pos:r.pos+n])
	r.pos += n

	return n, nil
}

func TestStalledReader_Read_ErrorHandling(t *testing.T) {
	protection := NewStalledStreamProtection(5*time.Minute, 1024)
	reader := &errorReader{err: errors.New("read error")}
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 10)

	_, err := wrapped.Read(buf)

	if err == nil {
		t.Error("Expected error from underlying reader")
	}
	if err.Error() != "read error" {
		t.Errorf("Expected 'read error', got %v", err)
	}
}

// errorReader simulates a reader that returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestStalledReader_Read_MultipleReads(t *testing.T) {
	protection := NewStalledStreamProtection(5*time.Minute, 0)
	data := "hello world test data"
	reader := strings.NewReader(data)
	ctx := context.Background()

	wrapped := protection.WrapReader(ctx, reader)
	buf := make([]byte, 5)

	var result strings.Builder
	for {
		n, err := wrapped.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			break
		}
	}

	if result.String() != data {
		t.Errorf("Expected %q, got %q", data, result.String())
	}
}
