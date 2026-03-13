package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

// gaugeCall records a single Gauge invocation.
type gaugeCall struct {
	name  string
	value float64
	tags  []string
}

// mockStatsd implements statsd.ClientInterface for testing.
type mockStatsd struct {
	calls []gaugeCall
}

func (m *mockStatsd) Gauge(name string, value float64, tags []string, _ float64) error {
	m.calls = append(m.calls, gaugeCall{name, value, tags})
	return nil
}
func (m *mockStatsd) GaugeWithTimestamp(_ string, _ float64, _ []string, _ float64, _ time.Time) error {
	return nil
}
func (m *mockStatsd) Count(_ string, _ int64, _ []string, _ float64) error { return nil }
func (m *mockStatsd) CountWithTimestamp(_ string, _ int64, _ []string, _ float64, _ time.Time) error {
	return nil
}
func (m *mockStatsd) Histogram(_ string, _ float64, _ []string, _ float64) error    { return nil }
func (m *mockStatsd) Distribution(_ string, _ float64, _ []string, _ float64) error { return nil }
func (m *mockStatsd) Set(_ string, _ string, _ []string, _ float64) error           { return nil }
func (m *mockStatsd) Timing(_ string, _ time.Duration, _ []string, _ float64) error { return nil }
func (m *mockStatsd) TimeInMilliseconds(_ string, _ float64, _ []string, _ float64) error {
	return nil
}
func (m *mockStatsd) Event(_ *statsd.Event) error                                    { return nil }
func (m *mockStatsd) SimpleEvent(_, _ string) error                                  { return nil }
func (m *mockStatsd) ServiceCheck(_ *statsd.ServiceCheck) error                      { return nil }
func (m *mockStatsd) SimpleServiceCheck(_ string, _ statsd.ServiceCheckStatus) error { return nil }
func (m *mockStatsd) Close() error                                                   { return nil }
func (m *mockStatsd) Flush() error                                                   { return nil }
func (m *mockStatsd) SetWriteTimeout(_ time.Duration) error                          { return nil }
func (m *mockStatsd) Incr(_ string, _ []string, _ float64) error                     { return nil }
func (m *mockStatsd) Decr(_ string, _ []string, _ float64) error                     { return nil }
func (m *mockStatsd) GetTelemetry() statsd.Telemetry                                 { return statsd.Telemetry{} }
func (m *mockStatsd) IsClosed() bool                                                 { return false }

// findGauge returns true if a gauge with the given name was recorded.
func (m *mockStatsd) findGauge(name string) (gaugeCall, bool) {
	for _, c := range m.calls {
		if c.name == name {
			return c, true
		}
	}
	return gaugeCall{}, false
}

func loadTestXML(t *testing.T, path string) passengerStatus {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var r io.Reader = bytes.NewReader(data)
	ps, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	return ps
}

func TestChartPoolUse(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	chartPoolUse(&ps, mock, nil, false)

	if c, ok := mock.findGauge("passenger.pool.used"); !ok || c.value != 3 {
		t.Errorf("passenger.pool.used: got %v, want 3", c.value)
	}
	if c, ok := mock.findGauge("passenger.pool.max"); !ok || c.value != 15 {
		t.Errorf("passenger.pool.max: got %v, want 15", c.value)
	}
}

func TestChartPoolUseWithTags(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	chartPoolUse(&ps, mock, []string{"source:test", "service:my-service"}, false)

	c, ok := mock.findGauge("passenger.pool.used")
	if !ok {
		t.Fatal("passenger.pool.used not recorded")
	}
	if len(c.tags) != 2 || c.tags[0] != "source:test" || c.tags[1] != "service:my-service" {
		t.Errorf("passenger.pool.used tags: got %v, want [source:test service:my-service]", c.tags)
	}
}

func TestTagsNil(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	// nil tags must not panic
	chartPoolUse(&ps, mock, nil, false)
	chartProcessed(&ps, mock, nil, false)
	chartMemory(&ps, mock, nil, false)
	chartPendingRequest(&ps, mock, nil, false)
	chartProcessUptime(&ps, mock, nil, false)
	chartProcessUse(&ps, mock, nil, false)
}

func TestChartProcessed(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	chartProcessed(&ps, mock, nil, false)

	// data.xml: processed values [4090, 20, 2] → sum=4112, min=2, max=4090, avg=1370
	if c, ok := mock.findGauge("passenger.processed.total"); !ok || c.value != 4112 {
		t.Errorf("passenger.processed.total: got %v, want 4112", c.value)
	}
	if c, ok := mock.findGauge("passenger.processed.min"); !ok || c.value != 2 {
		t.Errorf("passenger.processed.min: got %v, want 2", c.value)
	}
	if c, ok := mock.findGauge("passenger.processed.max"); !ok || c.value != 4090 {
		t.Errorf("passenger.processed.max: got %v, want 4090", c.value)
	}
}

func TestChartMemory(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	chartMemory(&ps, mock, nil, false)

	// data.xml: real_memory [326500, 372428, 416272] → sum=1115200 KB → /1024=1089 MB
	if c, ok := mock.findGauge("passenger.memory.total"); !ok || c.value != float64(1115200/1024) {
		t.Errorf("passenger.memory.total: got %v, want %v", c.value, float64(1115200/1024))
	}
}

func TestChartPendingRequest(t *testing.T) {
	ps := loadTestXML(t, "sample_data/dw_74.xml")
	mock := &mockStatsd{}
	chartPendingRequest(&ps, mock, nil, false)

	if c, ok := mock.findGauge("passenger.queue.depth"); !ok || c.value != 53 {
		t.Errorf("passenger.queue.depth: got %v, want 53", c.value)
	}
}

func TestChartDiscreteMetrics(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")

	// Replace execCommand so getProcessThreadCount returns "4" for any pid.
	oldCmd := execCommand
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "4")
	}
	defer func() { execCommand = oldCmd }()

	mock := &mockStatsd{}
	chartDiscreteMetrics(&ps, mock, nil, false)

	// Expect one thread-count gauge per process (3 processes in data.xml).
	threadGauges := 0
	for _, c := range mock.calls {
		if c.name == "passenger.process.threads" {
			threadGauges++
			if c.value != 4 {
				t.Errorf("passenger.process.threads: got %v, want 4", c.value)
			}
		}
	}
	if threadGauges != 3 {
		t.Errorf("expected 3 thread gauges, got %d", threadGauges)
	}

	// Expect memory and idle-time gauges for each process too.
	memGauges := 0
	for _, c := range mock.calls {
		if c.name == "passenger.process.memory" {
			memGauges++
		}
	}
	if memGauges != 3 {
		t.Errorf("expected 3 memory gauges, got %d", memGauges)
	}
}

func TestChartDiscreteMetricsWithTags(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")

	oldCmd := execCommand
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "4")
	}
	defer func() { execCommand = oldCmd }()

	mock := &mockStatsd{}
	chartDiscreteMetrics(&ps, mock, []string{"source:test", "service:my-service"}, false)

	for _, c := range mock.calls {
		if c.name != "passenger.process.threads" {
			continue
		}
		hasPid := false
		hasSource := false
		hasService := false
		for _, tag := range c.tags {
			if len(tag) > 4 && tag[:4] == "pid:" {
				hasPid = true
			}
			if tag == "source:test" {
				hasSource = true
			}
			if tag == "service:my-service" {
				hasService = true
			}
		}
		if !hasPid || !hasSource || !hasService {
			t.Errorf("passenger.process.threads tags missing expected values: got %v", c.tags)
		}
	}
}
