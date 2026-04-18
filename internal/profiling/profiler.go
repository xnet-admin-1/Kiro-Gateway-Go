// Package profiling provides CPU and memory profiling utilities for performance optimization.
package profiling

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// Profiler manages CPU and memory profiling operations.
type Profiler struct {
	cpuFile    *os.File
	memFile    *os.File
	profiling  bool
	startTime  time.Time
	outputDir  string
}

// NewProfiler creates a new profiler instance.
func NewProfiler(outputDir string) *Profiler {
	if outputDir == "" {
		outputDir = "."
	}
	return &Profiler{
		outputDir: outputDir,
	}
}

// StartCPUProfile starts CPU profiling.
func (p *Profiler) StartCPUProfile() error {
	if p.profiling {
		return fmt.Errorf("profiling already started")
	}

	filename := fmt.Sprintf("%s/cpu_profile_%d.prof", p.outputDir, time.Now().Unix())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	p.cpuFile = f
	p.profiling = true
	p.startTime = time.Now()
	return nil
}

// StopCPUProfile stops CPU profiling.
func (p *Profiler) StopCPUProfile() error {
	if !p.profiling || p.cpuFile == nil {
		return fmt.Errorf("CPU profiling not started")
	}

	pprof.StopCPUProfile()
	err := p.cpuFile.Close()
	p.cpuFile = nil
	p.profiling = false
	return err
}

// WriteMemProfile writes a memory profile.
func (p *Profiler) WriteMemProfile() error {
	filename := fmt.Sprintf("%s/mem_profile_%d.prof", p.outputDir, time.Now().Unix())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer f.Close()

	runtime.GC() // Force garbage collection before profiling
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	return nil
}

// ProfileFunction profiles a function execution.
func (p *Profiler) ProfileFunction(ctx context.Context, name string, fn func() error) (*ProfileResult, error) {
	// Start CPU profiling
	if err := p.StartCPUProfile(); err != nil {
		return nil, err
	}

	// Record memory stats before
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Record memory stats after
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Stop CPU profiling
	if stopErr := p.StopCPUProfile(); stopErr != nil {
		return nil, stopErr
	}

	// Write memory profile
	if memErr := p.WriteMemProfile(); memErr != nil {
		return nil, memErr
	}

	result := &ProfileResult{
		Name:           name,
		Duration:       duration,
		AllocsBefore:   memBefore.Alloc,
		AllocsAfter:    memAfter.Alloc,
		AllocsDelta:    memAfter.Alloc - memBefore.Alloc,
		GCCycles:       memAfter.NumGC - memBefore.NumGC,
		Error:          err,
	}

	return result, nil
}

// ProfileResult contains profiling results for a function execution.
type ProfileResult struct {
	Name         string
	Duration     time.Duration
	AllocsBefore uint64
	AllocsAfter  uint64
	AllocsDelta  uint64
	GCCycles     uint32
	Error        error
}

// String returns a formatted string representation of the profile result.
func (r *ProfileResult) String() string {
	status := "SUCCESS"
	if r.Error != nil {
		status = fmt.Sprintf("ERROR: %v", r.Error)
	}

	return fmt.Sprintf(
		"Profile: %s\n"+
			"  Status: %s\n"+
			"  Duration: %v\n"+
			"  Memory Delta: %d bytes\n"+
			"  GC Cycles: %d\n",
		r.Name, status, r.Duration, r.AllocsDelta, r.GCCycles,
	)
}

// IsPerformant checks if the result meets performance criteria.
func (r *ProfileResult) IsPerformant(maxDuration time.Duration, maxMemory uint64) bool {
	return r.Duration <= maxDuration && r.AllocsDelta <= maxMemory
}
