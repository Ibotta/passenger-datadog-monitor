package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"reflect"
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

// countCall records a single Count invocation.
type countCall struct {
	name  string
	value int64
	tags  []string
}

// histogramCall records a single Histogram invocation.
type histogramCall struct {
	name  string
	value float64
	tags  []string
}

// mockStatsd implements statsd.ClientInterface for testing.
type mockStatsd struct {
	calls          []gaugeCall
	countCalls     []countCall
	histogramCalls []histogramCall
}

func (m *mockStatsd) Gauge(name string, value float64, tags []string, _ float64) error {
	m.calls = append(m.calls, gaugeCall{name, value, tags})
	return nil
}
func (m *mockStatsd) GaugeWithTimestamp(_ string, _ float64, _ []string, _ float64, _ time.Time) error {
	return nil
}
func (m *mockStatsd) Count(name string, value int64, tags []string, _ float64) error {
	m.countCalls = append(m.countCalls, countCall{name, value, tags})
	return nil
}
func (m *mockStatsd) CountWithTimestamp(_ string, _ int64, _ []string, _ float64, _ time.Time) error {
	return nil
}
func (m *mockStatsd) Histogram(name string, value float64, tags []string, _ float64) error {
	m.histogramCalls = append(m.histogramCalls, histogramCall{name, value, tags})
	return nil
}
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

// findCount returns the first count call with the given name.
func (m *mockStatsd) findCount(name string) (countCall, bool) {
	for _, c := range m.countCalls {
		if c.name == name {
			return c, true
		}
	}
	return countCall{}, false
}

// countHistograms returns the number of histogram calls with the given name.
func (m *mockStatsd) countHistograms(name string) int {
	n := 0
	for _, h := range m.histogramCalls {
		if h.name == name {
			n++
		}
	}
	return n
}

func TestParseTags(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"source:foo,env:bar", []string{"source:foo", "env:bar"}},
		{"source:foo env:bar", []string{"source:foo", "env:bar"}},
		{"source:foo, env:bar", []string{"source:foo", "env:bar"}},
		{"version:1.2.3 service:my-service env:production source:my-service", []string{"version:1.2.3", "service:my-service", "env:production", "source:my-service"}},
		{"single", []string{"single"}},
	}
	for _, tc := range cases {
		got := parseTags(tc.input)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
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
	tracker := newDeltaTracker()
	// nil tags must not panic
	chartPoolUse(&ps, mock, nil, false)
	chartProcessed(&ps, mock, nil, false, tracker)
	chartMemory(&ps, mock, nil, false)
	chartPendingRequest(&ps, mock, nil, false)
	chartProcessUptime(&ps, mock, nil, false)
	chartProcessUse(&ps, mock, nil, false)
	chartDiscreteMetrics(&ps, mock, nil, false)
}

func TestChartProcessed(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	tracker := newDeltaTracker()

	// First scrape: histograms sent per process, no count (no previous value yet).
	mock1 := &mockStatsd{}
	chartProcessed(&ps, mock1, nil, false, tracker)

	if n := mock1.countHistograms("passenger.processed"); n != 3 {
		t.Errorf("expected 3 histogram calls for passenger.processed, got %d", n)
	}
	// No pid: tag on histogram calls.
	for _, h := range mock1.histogramCalls {
		if h.name != "passenger.processed" {
			continue
		}
		for _, tag := range h.tags {
			if len(tag) > 4 && tag[:4] == "pid:" {
				t.Errorf("passenger.processed histogram must not have pid tag, got %v", h.tags)
			}
		}
	}
	if len(mock1.countCalls) != 0 {
		t.Errorf("expected 0 count calls on first scrape, got %d", len(mock1.countCalls))
	}

	// Second scrape with same data: count delta = 0.
	mock2 := &mockStatsd{}
	chartProcessed(&ps, mock2, nil, false, tracker)

	if c, ok := mock2.findCount("passenger.processed.total"); !ok || c.value != 0 {
		t.Errorf("passenger.processed.total count delta: got %v, want 0", c.value)
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

	// Expect one histogram call per process (3 processes).
	if n := mock.countHistograms("passenger.memory"); n != 3 {
		t.Errorf("expected 3 histogram calls for passenger.memory, got %d", n)
	}

	// No pid: tag on histogram calls.
	for _, h := range mock.histogramCalls {
		if h.name != "passenger.memory" {
			continue
		}
		for _, tag := range h.tags {
			if len(tag) > 4 && tag[:4] == "pid:" {
				t.Errorf("passenger.memory histogram must not have pid tag, got %v", h.tags)
			}
		}
	}

	// avg/min/max gauges must not be sent.
	for _, name := range []string{"passenger.memory.avg", "passenger.memory.min", "passenger.memory.max"} {
		if _, ok := mock.findGauge(name); ok {
			t.Errorf("%s gauge should not be sent", name)
		}
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

func TestChartProcessUptime(t *testing.T) {
	ps := loadTestXML(t, "sample_data/data.xml")
	mock := &mockStatsd{}
	chartProcessUptime(&ps, mock, nil, false)

	// Expect one histogram call per process (3 processes in data.xml).
	if n := mock.countHistograms("passenger.uptime"); n != 3 {
		t.Errorf("expected 3 histogram calls for passenger.uptime, got %d", n)
	}

	// uptime.avg/min/max gauges must not be sent.
	for _, name := range []string{"passenger.uptime.avg", "passenger.uptime.min", "passenger.uptime.max"} {
		if _, ok := mock.findGauge(name); ok {
			t.Errorf("%s gauge should not be sent", name)
		}
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

	// Expect one thread-count histogram per process (3 processes in data.xml).
	if n := mock.countHistograms("passenger.process.threads"); n != 3 {
		t.Errorf("expected 3 histogram calls for passenger.process.threads, got %d", n)
	}
	for _, h := range mock.histogramCalls {
		if h.name == "passenger.process.threads" && h.value != 4 {
			t.Errorf("passenger.process.threads: got %v, want 4", h.value)
		}
	}

	// Expect one last_used histogram per process.
	if n := mock.countHistograms("passenger.process.last_used"); n != 3 {
		t.Errorf("expected 3 histogram calls for passenger.process.last_used, got %d", n)
	}

	// passenger.process.memory and passenger.process.request_processed must not be emitted.
	if n := mock.countHistograms("passenger.process.memory"); n != 0 {
		t.Errorf("passenger.process.memory should not be emitted, got %d calls", n)
	}
	if len(mock.countCalls) != 0 {
		t.Errorf("expected no count calls, got %d", len(mock.countCalls))
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

	for _, h := range mock.histogramCalls {
		if h.name != "passenger.process.threads" {
			continue
		}
		hasPid := false
		hasSource := false
		hasService := false
		for _, tag := range h.tags {
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
		if hasPid {
			t.Errorf("passenger.process.threads must not have pid tag, got %v", h.tags)
		}
		if !hasSource || !hasService {
			t.Errorf("passenger.process.threads missing source/service tags: got %v", h.tags)
		}
	}
}

func TestDeltaTracker(t *testing.T) {
	tracker := newDeltaTracker()
	mock := &mockStatsd{}

	// First call: no count sent (no previous value).
	tracker.CountDelta(mock, "test.metric", 100, nil, 1)
	if len(mock.countCalls) != 0 {
		t.Errorf("expected no count on first call, got %d", len(mock.countCalls))
	}

	// Second call: delta = 150 - 100 = 50.
	tracker.CountDelta(mock, "test.metric", 150, nil, 1)
	if len(mock.countCalls) != 1 {
		t.Fatalf("expected 1 count call, got %d", len(mock.countCalls))
	}
	if mock.countCalls[0].value != 50 {
		t.Errorf("expected delta 50, got %d", mock.countCalls[0].value)
	}

	// Process restart: current < prev → send current as delta.
	tracker.CountDelta(mock, "test.metric", 10, nil, 1)
	if len(mock.countCalls) != 2 {
		t.Fatalf("expected 2 count calls, got %d", len(mock.countCalls))
	}
	if mock.countCalls[1].value != 10 {
		t.Errorf("expected delta 10 on restart, got %d", mock.countCalls[1].value)
	}

	// Different tags → separate tracking key; first call for that key, no count.
	tracker.CountDelta(mock, "test.metric", 200, []string{"pid:123"}, 1)
	if len(mock.countCalls) != 2 {
		t.Errorf("expected no new count for new tag key on first call, got %d", len(mock.countCalls))
	}
	tracker.CountDelta(mock, "test.metric", 250, []string{"pid:123"}, 1)
	if len(mock.countCalls) != 3 {
		t.Fatalf("expected 3 count calls, got %d", len(mock.countCalls))
	}
	if mock.countCalls[2].value != 50 {
		t.Errorf("expected delta 50 for pid:123, got %d", mock.countCalls[2].value)
	}
}
