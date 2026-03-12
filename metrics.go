package main

import (
	"fmt"
	"log"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

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

func processPerThreadRequests(passengerDetails *passengerStatus) map[int]float64 {
	result := make(map[int]float64)
	for _, p := range passengerDetails.Processes {
		result[p.PID] = float64(p.Processed)
	}
	return result
}

func chartPendingRequest(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	var totalQueued int
	for _, queued := range passengerDetails.QueuedCount {
		totalQueued += queued
	}
	if printOutput {
		fmt.Printf("\n|=====Queue Depth====|\n Queue Depth %d", totalQueued)
	}
	_ = client.Gauge("passenger.queue.depth", float64(totalQueued), nil, 1)
}

func chartPoolUse(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	if printOutput {
		fmt.Printf("\n|=====Pool Usage====|\n Used Pool %d\n Max Pool %d", passengerDetails.ProcessCount, passengerDetails.PoolMax)
	}
	_ = client.Gauge("passenger.pool.used", float64(passengerDetails.ProcessCount), nil, 1)
	_ = client.Gauge("passenger.pool.max", float64(passengerDetails.PoolMax), nil, 1)
}

func chartProcessed(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	stats := processed(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Processed====|\n Total processed %d\n Average processed %d\n"+
			" Minimum processed %d\n Maximum processed %d", stats.sum, stats.avg, stats.min, stats.max)
	}
	_ = client.Gauge("passenger.processed.total", float64(stats.sum), nil, 1)
	_ = client.Gauge("passenger.processed.avg", float64(stats.avg), nil, 1)
	_ = client.Gauge("passenger.processed.min", float64(stats.min), nil, 1)
	_ = client.Gauge("passenger.processed.max", float64(stats.max), nil, 1)
}

func chartMemory(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	stats := memory(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Memory====|\n Total memory %d\n Average memory %d\n"+
			" Minimum memory %d\n Maximum memory %d", stats.sum/1024, stats.avg/1024, stats.min/1024, stats.max/1024)
	}
	_ = client.Gauge("passenger.memory.total", float64(stats.sum/1024), nil, 1)
	_ = client.Gauge("passenger.memory.avg", float64(stats.avg/1024), nil, 1)
	_ = client.Gauge("passenger.memory.min", float64(stats.min/1024), nil, 1)
	_ = client.Gauge("passenger.memory.max", float64(stats.max/1024), nil, 1)
}

func chartProcessUptime(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	stats := processUptime(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Process uptime====|\n Average uptime %d min\n"+
			" Minimum uptime %d min\n Maximum uptime %d min\n", stats.avg, stats.min, stats.max)
	}
	_ = client.Gauge("passenger.uptime.avg", float64(stats.avg), nil, 1)
	_ = client.Gauge("passenger.uptime.min", float64(stats.min), nil, 1)
	_ = client.Gauge("passenger.uptime.max", float64(stats.max), nil, 1)
}

func chartProcessUse(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	totalUsed := processUse(passengerDetails)
	if printOutput {
		fmt.Printf("\n|=====Process Usage====|\nUsed Processes %d", totalUsed)
	}
	_ = client.Gauge("passenger.processes.used", float64(totalUsed), nil, 1)
}

func chartDiscreteMetrics(passengerDetails *passengerStatus, client statsd.ClientInterface, printOutput bool) {
	threadCounts := processSystemThreadUsage(passengerDetails)
	memoryUsages := processPerThreadMemoryUsage(passengerDetails)
	idleTimes := processPerThreadIdleTime(passengerDetails)
	requestCounts := processPerThreadRequests(passengerDetails)

	if printOutput {
		fmt.Println("\n|====Process Thread Counts====|")
	}
	for pid, count := range threadCounts {
		if printOutput {
			fmt.Printf("PID: %d  Running: %0.2f threads\n", pid, count)
		}
		_ = client.Gauge("passenger.process.threads", count, []string{fmt.Sprintf("pid:%d", pid)}, 1)
	}

	if printOutput {
		fmt.Println("|====Process Memory Usage====|")
	}
	for pid, memUse := range memoryUsages {
		if printOutput {
			fmt.Printf("PID: %d Memory_Used: %0.2f MB\n", pid, memUse)
		}
		_ = client.Gauge("passenger.process.memory", memUse, []string{fmt.Sprintf("pid:%d", pid)}, 1)
	}

	if printOutput {
		fmt.Println("|====Process Idle Times====|")
	}
	for pid, seconds := range idleTimes {
		if printOutput {
			fmt.Printf("PID: %d Idle: %d Seconds\n", pid, int(seconds))
		}
		_ = client.Gauge("passenger.process.last_used", seconds, []string{fmt.Sprintf("pid:%d", pid)}, 1)
	}

	if printOutput {
		fmt.Println("|====Process Requests Handled====|")
	}
	for pid, count := range requestCounts {
		if printOutput {
			fmt.Printf("PID: %d Processed: %d Requests\n", pid, int(count))
		}
		_ = client.Gauge("passenger.process.request_processed", count, []string{fmt.Sprintf("pid:%d", pid)}, 1)
	}
}
