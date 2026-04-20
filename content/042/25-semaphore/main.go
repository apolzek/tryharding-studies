package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const total = 12
	const max = 3

	var inFlight, peak atomic.Int64
	calls := make([]Caller, total)
	for i := range calls {
		calls[i] = APICaller(120*time.Millisecond, &inFlight, &peak)
	}

	start := time.Now()
	resps, errs := LimitedRun(ctx, max, calls)
	elapsed := time.Since(start)

	for i, r := range resps {
		if errs[i] != nil {
			fmt.Printf("call %d: erro=%v\n", i, errs[i])
			continue
		}
		fmt.Printf("call %d: %s\n", i, r)
	}
	fmt.Printf("pico de concorrência=%d limite=%d elapsed=%s\n",
		peak.Load(), max, elapsed.Round(time.Millisecond))
}
