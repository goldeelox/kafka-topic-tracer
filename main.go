package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/segmentio/kafka-go"
	"os"
	"os/signal"
	"time"
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

func isZero(d int64) bool {
	if d == 0 {
		return true
	}
	return false
}

func avgMessageSize(count, size int64) int64 {
	if isZero(count) || isZero(size) {
		return 0
	}

	return size / count
}

func messagesPerSec(start time.Time, count int64) float32 {
	secondsElapsed := time.Now().Unix() - start.Unix()
	if isZero(secondsElapsed) || isZero(count) {
		return 0
	}

	return float32(count) / float32(secondsElapsed)
}

func outputStats(start time.Time, count, size int64) {
	avg := avgMessageSize(count, size)
	mps := messagesPerSec(start, count)
	et := time.Now().Sub(start)
	fmt.Printf("  Elapsed time: %s, Messages recieved: %d (%.2f/sec), Average size: %d\033[0K\r", et.Round(time.Second).String(), count, mps, avg)
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
	start := time.Now()
	var totalCount, totalMsgSize int64

	ticker := time.NewTicker(intervalSec * time.Second)
	for {
		select {
		case <-ticker.C:
			stats := r.Stats()
			totalCount += stats.Messages
			totalMsgSize += stats.Bytes
			if isZero(stats.Offset) {
				// reset start timer until coordinator assigns partitions
				start = time.Now()
				continue
			}
			outputStats(start, totalCount, totalMsgSize)
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
