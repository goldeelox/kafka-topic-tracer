package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	topic string
  bootstrapServer string
  groupId string
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

func messageSize(m []byte) int {
	return len(m)
}

func avgMessageSize(count, size int) int {
	return size / count
}

func messagesPerSec(start time.Time, count int) float32 {
	c := int64(count)
	d := time.Now().Unix() - start.Unix()

	if d > 0 {
		return float32(c) / float32(d)
	}
	return 0
}

func outputStats(start time.Time, count, size int) {
	avg := avgMessageSize(count, size)
	mps := messagesPerSec(start, count)
  et := time.Now().Sub(start)
	fmt.Printf("\033[1A  Elapsed time: %s, Messages recieved: %d (%.2f/sec), Average size: %d\n", et.Round(time.Second).String(), count, mps, avg)
}

func topicPoller(c *kafka.Consumer) {
  fmt.Printf("  Waiting for messages...\n")
	start := time.Now()
	totalCount, totalMsgSize := 0, 0
	for {
		ev := c.Poll(1000)
    if totalCount > 0 && totalMsgSize > 0 {
      outputStats(start, totalCount, totalMsgSize)
    }
		switch e := ev.(type) {
		case *kafka.Message:
			totalCount += 1
			totalMsgSize += messageSize(e.Value)
			outputStats(start, totalCount, totalMsgSize)
		case kafka.Error:
			fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
			c.Close()
			os.Exit(1)
		default:
			// reset start time until we start consuming messages
			if totalCount == 0 {
				start = time.Now()
			}
		}
	}
}

func main() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServer,
		"group.id":           groupId,
		"auto.offset.reset":  "latest",
		"enable.auto.commit": false,
	})
	if err != nil {
		fmt.Printf("Error creating new consumer: %v", err)
		os.Exit(1)
	}

	err = consumer.Subscribe(topic, nil)
	if err != nil {
		fmt.Printf("Error subscribing to topic: %v", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

  fmt.Printf("Broker: %s, Topic: %s\n", bootstrapServer, topic)
	go topicPoller(consumer)

	<-c
  fmt.Printf("Unsubscribing and closing consumer connection...")
  err = consumer.Unassign()
	err = consumer.Unsubscribe()
	err = consumer.Close()
  if err != nil {
    fmt.Printf("Error closing session: %v", err)
  }
	os.Exit(0)
}
