package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Stat struct {
	Name  string
	Value string
}

type ReaderStats struct {
	StartTime    time.Time
	MessageCount int64
	MessageSize  int64
}

func isZero(d int64) bool {
	if d == 0 {
		return true
	}
	return false
}

func updateStatsOutput(stats []Stat) {
	var msg []string
	for _, stat := range stats {
		v := fmt.Sprintf("%s: %s", stat.Name, stat.Value)
		msg = append(msg, v)
	}
	msgString := strings.Join(msg, ", ")
	fmt.Printf("  %s\033[0K\r", msgString)
}

func (s *ReaderStats) countString() string {
	return strconv.FormatInt(s.MessageCount, 10)
}

func (s *ReaderStats) elapsedTime() time.Duration {
	return time.Now().Sub(s.StartTime).Round(time.Second)
}

func (s *ReaderStats) messageRateString() string {
	if isZero(s.MessageCount) {
		return "0"
	}
	rate := float64(s.MessageCount) / s.elapsedTime().Seconds()
	return strconv.FormatFloat(rate, 'f', 2, 64)
}

func (s *ReaderStats) avgMessageSizeString() string {
	if isZero(s.MessageCount) || isZero(s.MessageSize) {
		return "0"
	}
	return strconv.FormatInt(s.MessageSize/s.MessageCount, 10)
}
