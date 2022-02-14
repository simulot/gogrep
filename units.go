package main

import (
	"fmt"
	"math"
	"time"
)

type ByteCounter int64

func (b ByteCounter) String() string {
	u := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	n := float64(b)
	for math.Round(n/1000) >= 1 {
		n /= 1000
		i++
	}
	return fmt.Sprintf("%.1f%s", n, u[i])
}

type BytePerSecond struct {
	B int64
	D time.Duration
}

func (bps BytePerSecond) String() string {
	return ByteCounter(float64(bps.B)/bps.D.Seconds()).String() + "/s"
}
