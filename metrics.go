package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

// deltaTracker tracks previous values for monotonically increasing counters
// to send deltas instead of absolute values.
type deltaTracker struct {
	prev map[string]int64
}

func newDeltaTracker() *deltaTracker {
	return &deltaTracker{prev: make(map[string]int64)}
}

// CountDelta sends a Count metric with the delta since the last call.
// On the first call for a given metric+tags key, no count is sent (to avoid
// a spike from the initial absolute value). If the delta is negative
// (e.g. process restart), the current value is sent as the delta.
func (d *deltaTracker) CountDelta(client statsd.ClientInterface, name string, current int64, tags []string, rate float64) {
	sorted := make([]string, len(tags))
	copy(sorted, tags)
	sort.Strings(sorted)
	key := name + ":" + strings.Join(sorted, ",")
	if prev, ok := d.prev[key]; ok {
		delta := current - prev
		if delta < 0 {
			delta = current
		}
		_ = client.Count(name, delta, tags, rate)
	}
	d.prev[key] = current
}

func processSystemThreadUsage(passengerDetails *passengerStatus) map[int]float64 {
	processThreads := make(map[int]float64)
	for _, p := range passengerDetails.Processes {
		tc, err := getProcessThreadCount(p.PID)
		if err != nil {
			log.Printf("encountered error getting thread count: %s", err)
		}
		processThreads[p.PID] = float64(tc)
	}
	return processThreads
}

func processPerThreadMemoryUsage(passengerDetails *passengerStatus) map[int]float64 {
	result := make(map[int]float64)
	for _, p := range passengerDetails.Processes {
		result[p.PID] = float64(p.Memory) / 1024
	}
	return result
}

// processPerThreadIdleTime calculates seconds since last use per process.
// Passenger timestamps are in microseconds; multiply by 1000 to convert to nanoseconds.
func processPerThreadIdleTime(passengerDetails *passengerStatus) map[int]float64 {
	result := make(map[int]float64)
	for _, p := range passengerDetails.Processes {
		lastUsedTime := time.Unix(0, p.LastUsed*1000)
		result[p.PID] = time.Since(lastUsedTime).Seconds()
	}
	return result
}

func chartPendingRequest(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool) {
	var totalQueued int
	for _, queued := range passengerDetails.QueuedCount {
		totalQueued += queued
	}
	if printOutput {
		fmt.Printf("\n|=====Queue Depth====|\n Queue Depth %d", totalQueued)
	}
	_ = client.Gauge("passenger.queue.depth", float64(totalQueued), tags, 1)
}

func chartPoolUse(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool) {
	if printOutput {
		fmt.Printf("\n|=====Pool Usage====|\n Used Pool %d\n Max Pool %d", passengerDetails.ProcessCount, passengerDetails.PoolMax)
	}
	_ = client.Gauge("passenger.pool.used", float64(passengerDetails.ProcessCount), tags, 1)
	_ = client.Gauge("passenger.pool.max", float64(passengerDetails.PoolMax), tags, 1)
}

func chartProcessed(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool, tracker *deltaTracker) {
	var totalProcessed int64
	for _, p := range passengerDetails.Processes {
		totalProcessed += int64(p.Processed)
		_ = client.Histogram("passenger.processed", float64(p.Processed), pidTags(tags, p.PID), 1)
	}
	if printOutput {
		fmt.Printf("\n|=====Processed====|\n Total processed %d", totalProcessed)
	}
	tracker.CountDelta(client, "passenger.processed.total", totalProcessed, tags, 1)
}

func chartMemory(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool) {
	var totalMemoryKB int64
	for _, p := range passengerDetails.Processes {
		totalMemoryKB += int64(p.Memory)
		_ = client.Histogram("passenger.memory", float64(p.Memory)/1024, pidTags(tags, p.PID), 1)
	}
	totalMB := float64(totalMemoryKB / 1024)
	if printOutput {
		fmt.Printf("\n|=====Memory====|\n Total memory %d MB", int(totalMB))
	}
	_ = client.Gauge("passenger.memory.total", totalMB, tags, 1)
}

func chartProcessUptime(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool) {
	stats := processUptime(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Process uptime====|\n Average uptime %d min\n"+
			" Minimum uptime %d min\n Maximum uptime %d min\n", stats.avg, stats.min, stats.max)
	}
	_ = client.Gauge("passenger.uptime.avg", float64(stats.avg), tags, 1)
	_ = client.Gauge("passenger.uptime.min", float64(stats.min), tags, 1)
	_ = client.Gauge("passenger.uptime.max", float64(stats.max), tags, 1)
}

func chartProcessUse(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool) {
	totalUsed := processUse(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Process Usage====|\nUsed Processes %d", totalUsed)
	}
	_ = client.Gauge("passenger.processes.used", float64(totalUsed), tags, 1)
}

func pidTags(baseTags []string, pid int) []string {
	t := make([]string, len(baseTags), len(baseTags)+1)
	copy(t, baseTags)
	return append(t, fmt.Sprintf("pid:%d", pid))
}

func chartDiscreteMetrics(passengerDetails *passengerStatus, client statsd.ClientInterface, tags []string, printOutput bool, tracker *deltaTracker) {
	threadCounts := processSystemThreadUsage(passengerDetails)
	memoryUsages := processPerThreadMemoryUsage(passengerDetails)
	idleTimes := processPerThreadIdleTime(passengerDetails)

	if printOutput {
		fmt.Println("\n|====Process Thread Counts====|")
	}
	for pid, count := range threadCounts {
		if printOutput {
			fmt.Printf("PID: %d  Running: %0.2f threads\n", pid, count)
		}
		_ = client.Gauge("passenger.process.threads", count, pidTags(tags, pid), 1)
	}

	if printOutput {
		fmt.Println("|====Process Memory Usage====|")
	}
	for pid, memUse := range memoryUsages {
		if printOutput {
			fmt.Printf("PID: %d Memory_Used: %0.2f MB\n", pid, memUse)
		}
		_ = client.Histogram("passenger.process.memory", memUse, pidTags(tags, pid), 1)
	}

	if printOutput {
		fmt.Println("|====Process Idle Times====|")
	}
	for pid, seconds := range idleTimes {
		if printOutput {
			fmt.Printf("PID: %d Idle: %d Seconds\n", pid, int(seconds))
		}
		_ = client.Gauge("passenger.process.last_used", seconds, pidTags(tags, pid), 1)
	}

	if printOutput {
		fmt.Println("|====Process Requests Handled====|")
	}
	for _, p := range passengerDetails.Processes {
		if printOutput {
			fmt.Printf("PID: %d Processed: %d Requests\n", p.PID, p.Processed)
		}
		tracker.CountDelta(client, "passenger.process.request_processed", int64(p.Processed), pidTags(tags, p.PID), 1)
	}
}
