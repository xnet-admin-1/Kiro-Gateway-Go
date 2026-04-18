package profiling

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProfiler_StartStopCPUProfile(t *testing.T) {
	profiler := NewProfiler(".")
	
	// Test starting CPU profile
	err := profiler.StartCPUProfile()
	if err != nil {
		t.Fatalf("Failed to start CPU profile: %v", err)
	}
	
	// Test double start (should fail)
	err = profiler.StartCPUProfile()
	if err == nil {
		t.Error("Expected error when starting CPU profile twice")
	}
	
	// Test stopping CPU profile
	err = profiler.StopCPUProfile()
	if err != nil {
		t.Fatalf("Failed to stop CPU profile: %v", err)
	}
	
	// Test double stop (should fail)
	err = profiler.StopCPUProfile()
	if err == nil {
		t.Error("Expected error when stopping CPU profile twice")
	}
}

func TestProfiler_WriteMemProfile(t *testing.T) {
	profiler := NewProfiler(".")
	
	err := profiler.WriteMemProfile()
	if err != nil {
		t.Fatalf("Failed to write memory profile: %v", err)
	}
}

func TestProfiler_ProfileFunction(t *testing.T) {
	profiler := NewProfiler(".")
	
	// Test function that allocates memory
	testFunc := func() error {
		// Allocate some memory
		data := make([]byte, 1024*1024) // 1MB
		_ = data
		time.Sleep(10 * time.Millisecond)
		return nil
	}
	
	result, err := profiler.ProfileFunction(context.Background(), "test_function", testFunc)
	if err != nil {
		t.Fatalf("Failed to profile function: %v", err)
	}
	
	if result.Name != "test_function" {
		t.Errorf("Expected name 'test_function', got %s", result.Name)
	}
	
	if result.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", result.Duration)
	}
	
	if result.AllocsDelta == 0 {
		t.Error("Expected memory allocation delta > 0")
	}
	
	t.Logf("Profile result: %s", result.String())
}

func TestProfileResult_IsPerformant(t *testing.T) {
	tests := []struct {
		name        string
		result      *ProfileResult
		maxDuration time.Duration
		maxMemory   uint64
		want        bool
	}{
		{
			name: "performant",
			result: &ProfileResult{
				Duration:    5 * time.Millisecond,
				AllocsDelta: 1024,
			},
			maxDuration: 10 * time.Millisecond,
			maxMemory:   2048,
			want:        true,
		},
		{
			name: "slow",
			result: &ProfileResult{
				Duration:    15 * time.Millisecond,
				AllocsDelta: 1024,
			},
			maxDuration: 10 * time.Millisecond,
			maxMemory:   2048,
			want:        false,
		},
		{
			name: "memory hungry",
			result: &ProfileResult{
				Duration:    5 * time.Millisecond,
				AllocsDelta: 4096,
			},
			maxDuration: 10 * time.Millisecond,
			maxMemory:   2048,
			want:        false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.IsPerformant(tt.maxDuration, tt.maxMemory)
			if got != tt.want {
				t.Errorf("IsPerformant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewProfiler(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		want      string
	}{
		{
			name:      "default directory",
			outputDir: "",
			want:      ".",
		},
		{
			name:      "custom directory",
			outputDir: "/tmp",
			want:      "/tmp",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiler := NewProfiler(tt.outputDir)
			if profiler.outputDir != tt.want {
				t.Errorf("NewProfiler() outputDir = %v, want %v", profiler.outputDir, tt.want)
			}
		})
	}
}

// Cleanup function to remove profile files after tests
func TestMain(m *testing.M) {
	code := m.Run()
	
	// Clean up profile files
	files, _ := os.ReadDir(".")
	for _, file := range files {
		if len(file.Name()) > 5 && file.Name()[len(file.Name())-5:] == ".prof" {
			os.Remove(file.Name())
		}
	}
	
	os.Exit(code)
}
