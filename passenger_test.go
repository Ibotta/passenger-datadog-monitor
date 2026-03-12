package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func mustReadXML(t *testing.T, path string) io.Reader {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return bytes.NewReader(data)
}

func TestParsePassengerXML_DataXML(t *testing.T) {
	r := mustReadXML(t, "sample_data/data.xml")
	got, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ProcessCount != 3 {
		t.Errorf("ProcessCount: got %d, want 3", got.ProcessCount)
	}
	if got.PoolMax != 15 {
		t.Errorf("PoolMax: got %d, want 15", got.PoolMax)
	}
	if len(got.Processes) != 3 {
		t.Errorf("len(Processes): got %d, want 3", len(got.Processes))
	}
	if len(got.QueuedCount) != 1 || got.QueuedCount[0] != 0 {
		t.Errorf("QueuedCount: got %v, want [0]", got.QueuedCount)
	}
}

func TestParsePassengerXML_DW74(t *testing.T) {
	r := mustReadXML(t, "sample_data/dw_74.xml")
	got, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ProcessCount != 10 {
		t.Errorf("ProcessCount: got %d, want 10", got.ProcessCount)
	}
	if len(got.Processes) != 10 {
		t.Errorf("len(Processes): got %d, want 10", len(got.Processes))
	}
	if len(got.QueuedCount) != 1 || got.QueuedCount[0] != 53 {
		t.Errorf("QueuedCount: got %v, want [53]", got.QueuedCount)
	}
}

func TestParsePassengerXML_Restarted(t *testing.T) {
	r := mustReadXML(t, "sample_data/restarted.xml")
	got, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ProcessCount != 0 {
		t.Errorf("ProcessCount: got %d, want 0", got.ProcessCount)
	}
	if len(got.Processes) != 0 {
		t.Errorf("len(Processes): got %d, want 0", len(got.Processes))
	}
}

func TestSummarizeStats(t *testing.T) {
	vals := []int{10, 3, 7, 1, 9}
	got := summarizeStats(&vals)

	if got.min != 1 {
		t.Errorf("min: got %d, want 1", got.min)
	}
	if got.max != 10 {
		t.Errorf("max: got %d, want 10", got.max)
	}
	if got.sum != 30 {
		t.Errorf("sum: got %d, want 30", got.sum)
	}
	if got.avg != 6 {
		t.Errorf("avg: got %d, want 6", got.avg)
	}
	if got.len != 5 {
		t.Errorf("len: got %d, want 5", got.len)
	}
}

func TestProcessed(t *testing.T) {
	r := mustReadXML(t, "sample_data/data.xml")
	ps, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats := processed(&ps)
	// data.xml processes: 4090, 20, 2 → sorted [2, 20, 4090]
	if stats.min != 2 {
		t.Errorf("min: got %d, want 2", stats.min)
	}
	if stats.max != 4090 {
		t.Errorf("max: got %d, want 4090", stats.max)
	}
	if stats.sum != 4112 {
		t.Errorf("sum: got %d, want 4112", stats.sum)
	}
}

func TestMemory(t *testing.T) {
	r := mustReadXML(t, "sample_data/data.xml")
	ps, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats := memory(&ps)
	// data.xml real_memory: 326500, 372428, 416272
	if stats.min != 326500 {
		t.Errorf("min: got %d, want 326500", stats.min)
	}
	if stats.max != 416272 {
		t.Errorf("max: got %d, want 416272", stats.max)
	}
	if stats.sum != 1115200 {
		t.Errorf("sum: got %d, want 1115200", stats.sum)
	}
}
