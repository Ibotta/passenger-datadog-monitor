// Package main is the entry point for passenger-datadog-monitor.
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

const (
	// DefaultHost is 127.0.0.1 (localhost)
	DefaultHost = "127.0.0.1"

	// DefaultPort is 8125
	DefaultPort = 8125
)

func main() {
	hostName := flag.String("host", DefaultHost, "DogStatsD Host")
	portNum := flag.Int("port", DefaultPort, "DogStatsD UDP Port")
	printOutput := flag.Bool("print", false, "Print Outputs")
	tagsFlag := flag.String("tags", "", "Comma-separated tags to add to all metrics (e.g. source:my-service,service:my-service)")
	flag.Parse()

	// backwards compatibility: positional "print" argument
	if flag.NArg() > 0 && flag.Arg(0) == "print" {
		*printOutput = true
	}

	var baseTags []string
	if *tagsFlag != "" {
		baseTags = strings.FieldsFunc(*tagsFlag, func(r rune) bool {
			return r == ',' || r == ' '
		})
	}

	client, err := statsd.New(fmt.Sprintf("%s:%d", *hostName, *portNum))
	if err != nil {
		log.Fatal("Error establishing StatsD connection:", err)
	}

	if *printOutput {
		log.Println("Starting loop, sending to", *hostName, *portNum)
	}

	for {
		xmlData, err := retrievePassengerStats()
		if err != nil {
			client.Close() //nolint:errcheck,gosec
			log.Fatal("Error getting passenger data:", err)
		}

		passengerData, err := parsePassengerXML(&xmlData)
		if err != nil {
			client.Close() //nolint:errcheck,gosec
			log.Fatal("Error parsing passenger data:", err)
		}

		if passengerData.ProcessCount == 0 {
			log.Println("Passenger has not yet started any threads, will try again next loop")
		} else {
			chartProcessed(&passengerData, client, baseTags, *printOutput)
			chartMemory(&passengerData, client, baseTags, *printOutput)
			chartPendingRequest(&passengerData, client, baseTags, *printOutput)
			chartPoolUse(&passengerData, client, baseTags, *printOutput)
			chartProcessUptime(&passengerData, client, baseTags, *printOutput)
			chartProcessUse(&passengerData, client, baseTags, *printOutput)
			chartDiscreteMetrics(&passengerData, client, baseTags, *printOutput)
		}

		time.Sleep(10 * time.Second)
	}
}
