package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	topic           string
	bootstrapServer string
	groupId         string
	ctx             context.Context
)

func init() {
	flag.StringVar(&topic, "topic", "", "Name of topic to trace")
	flag.StringVar(&bootstrapServer, "bootstrap-server", "localhost:9092", "The Kafka server to connect to")
	flag.StringVar(&groupId, "group-id", "kafka-topic-tracer", "Consumer group.id")
	flag.Parse()

	if topic == "" {
		flag.Usage()
		os.Exit(0)
	}
}

func topicReader() *kafka.Reader {
	maxWait, _ := time.ParseDuration("1s")
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{bootstrapServer},
		GroupID:     groupId,
		Topic:       topic,
		StartOffset: kafka.LastOffset,
		MaxWait:     maxWait,
	})
}

func messagePoller(r *kafka.Reader) {
	fmt.Printf("Broker: %s, Topic: %s\n", bootstrapServer, topic)
	fmt.Printf("  Waiting for group assignment...\033[0K\r")

	ctxTimeout, _ := time.ParseDuration("1s")

	for {
		fetchCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
		defer cancel()
		_, err := r.FetchMessage(fetchCtx)
		if err != nil && err.Error() != "context deadline exceeded" {
			fmt.Printf("  Error fetching message: %v\033[0K\r", err)
		}
	}
}

func statPoller(r *kafka.Reader, intervalSec time.Duration) {
  readerStats := &ReaderStats{
    StartTime: time.Now(),
    MessageCount: 0,
    MessageSize: 0,
  }

	ticker := time.NewTicker(intervalSec * time.Second)
	for {
		select {
		case <-ticker.C:
			stats := r.Stats()
      readerStats.MessageCount += stats.Messages
      readerStats.MessageSize += stats.Bytes

			if isZero(stats.Offset) {
				// reset start timer until coordinator assigns partitions
				readerStats.StartTime = time.Now()
				continue
			}

			updateStatsOutput([]Stat{
				{
					Name:  "Elapsed time",
					Value: readerStats.elapsedTime().String(),
				},
				{
					Name:  "Messages received",
					Value: readerStats.countString(),
				},
				{
					Name:  "Message rate (per sec)",
					Value: readerStats.messageRateString(),
				},
        {
          Name: "Avg message size (bytes)",
          Value: readerStats.avgMessageSizeString(),
        },
			})
		}
	}
}

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)
	ctx = context.Background()

	reader := topicReader()
	go messagePoller(reader)
	go statPoller(reader, 1)

	select {
	case <-done:
		fmt.Printf("\nClosing reader...\n")

		if err := reader.Close(); err != nil {
			fmt.Printf("Error closing reader: %v\n", err)
		}
	}
}
