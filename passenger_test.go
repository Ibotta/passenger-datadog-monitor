package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func mustReadXML(t *testing.T, path string) io.Reader {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return bytes.NewReader(data)
}

func TestParsePassengerXML_ExampleXML(t *testing.T) {
	r := mustReadXML(t, "sample_data/example.xml")
	got, err := parsePassengerXML(&r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ProcessCount != 3 {
		t.Errorf("ProcessCount: got %d, want 3", got.ProcessCount)
	}
	if got.PoolMax != 3 {
		t.Errorf("PoolMax: got %d, want 3", got.PoolMax)
	}
	if len(got.Processes) != 3 {
		t.Errorf("len(Processes): got %d, want 3", len(got.Processes))
	}
	if len(got.QueuedCount) != 1 || got.QueuedCount[0] != 0 {
		t.Errorf("QueuedCount: got %v, want [0]", got.QueuedCount)
	}
	if got.Processes[0].PID != 12529 {
		t.Errorf("Processes[0].PID: got %d, want 12529", got.Processes[0].PID)
	}
	if got.Processes[0].Processed != 1854 {
		t.Errorf("Processes[0].Processed: got %d, want 1854", got.Processes[0].Processed)
	}
	if got.Processes[0].Memory != 484884 {
		t.Errorf("Processes[0].Memory: got %d, want 484884", got.Processes[0].Memory)
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

