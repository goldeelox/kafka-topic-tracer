# kafka-topic-tracer
Utility to live monitor a Kafka topic message rate/throughput.

## Usage
```bash
Usage of ./kafka-topic-tracer:
  -bootstrap-server string
    	The Kafka server to connect to (default "localhost:9092")
  -group-id string
    	Consumer group.id (default "kafka-topic-tracer")
  -topic string
    	Name of topic to trace
```

## Sample output
```bash
$ ./kafka-topic-tracer -topic test
Broker: localhost:9092, Topic: test
    Elapsed time: 27s, Messages recieved: 1000 (37.04/sec), Average size: 100
```
