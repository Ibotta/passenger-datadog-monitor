package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// execCommand is a package-level variable to allow test overrides.
var execCommand = exec.Command

type passengerStatus struct {
	XMLName      xml.Name  `xml:"info"`
	ProcessCount int       `xml:"process_count"`
	PoolMax      int       `xml:"max"`
	PoolCurrent  int       `xml:"capacity_used"`
	QueuedCount  []int     `xml:"supergroups>supergroup>group>get_wait_list_size"`
	Processes    []process `xml:"supergroups>supergroup>group>processes>process"`
}

type process struct {
	CurrentSessions int   `xml:"sessions"`
	Processed       int   `xml:"processed"`
	SpawnTime       int64 `xml:"spawn_end_time"`
	CPU             int   `xml:"cpu"`
	Memory          int   `xml:"real_memory"`
	PID             int   `xml:"pid"`
	LastUsed        int64 `xml:"last_used"`
}

// Stats stores aggregate statistics for a set of process values.
type Stats struct {
	min int
	len int
	avg int
	max int
	sum int
}

func summarizeStats(statsArray *[]int) Stats {
	var s Stats
	sum, count := 0, len(*statsArray)
	sort.Sort(sort.IntSlice(*statsArray))
	for _, v := range *statsArray {
		sum += v
	}
	sorted := *statsArray
	s.min = sorted[0]
	s.len = count
	s.avg = sum / count
	s.max = sorted[len(sorted)-1]
	s.sum = sum
	return s
}

func retrievePassengerStats() (io.Reader, error) {
	out, err := execCommand("passenger-status", "--show=xml").Output() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("passenger-status error: %w", err)
	}
	return bytes.NewReader(out), nil
}

func parsePassengerXML(xmlData *io.Reader) (passengerStatus, error) {
	var parsed passengerStatus
	dec := xml.NewDecoder(*xmlData)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&parsed); err != nil {
		return passengerStatus{}, err
	}
	return parsed, nil
}

func getProcessThreadCount(pid int) (int, error) {
	out, err := execCommand("ps", "--no-header", "-o", "nlwp", strconv.Itoa(pid)).Output() //nolint:gosec
	if err != nil {
		return 0, fmt.Errorf("error getting thread count for pid %d: %w", pid, err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("error parsing thread count: %w", err)
	}
	return count, nil
}

func processed(passengerDetails *passengerStatus) Stats {
	var vals []int
	for _, p := range passengerDetails.Processes {
		vals = append(vals, p.Processed)
	}
	return summarizeStats(&vals)
}

func memory(passengerDetails *passengerStatus) Stats {
	var vals []int
	for _, p := range passengerDetails.Processes {
		vals = append(vals, p.Memory)
	}
	return summarizeStats(&vals)
}

// processUptime calculates uptime stats for all processes.
// Passenger timestamps are in microseconds; multiply by 1000 to convert to nanoseconds.
func processUptime(passengerDetails *passengerStatus) Stats {
	var upTimes []int
	for _, p := range passengerDetails.Processes {
		spawnedNano := time.Unix(0, p.SpawnTime*1000)
		upTimes = append(upTimes, int(time.Since(spawnedNano).Minutes()))
	}
	return summarizeStats(&upTimes)
}

func processUse(passengerDetails *passengerStatus) int {
	var totalUsed int
	periodStart := time.Now().Add(-(10 * time.Second))
	for _, p := range passengerDetails.Processes {
		lastUsedNano := time.Unix(0, p.LastUsed*1000)
		if lastUsedNano.After(periodStart) {
			totalUsed++
		}
	}
	return totalUsed
}
