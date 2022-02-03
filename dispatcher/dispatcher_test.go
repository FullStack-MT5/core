package dispatcher_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/benchttp/runner/dispatcher"
)

func TestDo(t *testing.T) {
	t.Run("stop when maxIter is reached", func(t *testing.T) {
		const (
			numWorker = 1
			maxIter   = 10
			expIter   = 10
		)

		gotIter := 0

		dispatcher.New(numWorker).Do(context.Background(), maxIter, func() {
			gotIter++
		})

		if gotIter != expIter {
			t.Errorf("iterations: exp %d, got %d", expIter, gotIter)
		}
	})

	t.Run("stop on context timeout", func(t *testing.T) {
		const (
			timeout   = 100 * time.Millisecond
			interval  = timeout / 10
			numWorker = 1

			margin      = 25 * time.Millisecond // determined empirically
			maxDuration = timeout + margin
		)

		var (
			maxIter = int(interval.Milliseconds()) + 1 // should not be reached
			gotIter = 0
		)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		gotDuration := timeFunc(func() {
			dispatcher.New(numWorker).Do(ctx, maxIter, func() {
				gotIter++
				time.Sleep(interval)
			})
		})

		if gotDuration > maxDuration {
			t.Errorf(
				"context timeout duration: exp < %dms, got %dms",
				maxDuration.Milliseconds(), gotDuration.Milliseconds(),
			)
		}

		if gotIter >= maxIter {
			t.Errorf(
				"context timeout iterations: exp < %d, got %d",
				maxIter, gotIter,
			)
		}
	})

	t.Run("stop on context cancel", func(t *testing.T) {
		const (
			timeout   = 100 * time.Millisecond
			interval  = timeout / 10
			numWorker = 1

			margin      = 25 * time.Millisecond // determined empirically
			maxDuration = timeout + margin
		)

		var (
			maxIter = int(interval.Milliseconds()) + 1 // should not be reached
			gotIter = 0
		)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(timeout)
			cancel()
		}()

		gotDuration := timeFunc(func() {
			dispatcher.New(numWorker).Do(ctx, maxIter, func() {
				time.Sleep(interval)
			})
		})

		if gotDuration > maxDuration {
			t.Errorf(
				"context cancel duration: exp < %dms, got %dms",
				maxDuration.Milliseconds(), gotDuration.Milliseconds(),
			)
		}

		if gotIter >= maxIter {
			t.Errorf(
				"context timeout iterations: exp < %d, got %d",
				maxIter, gotIter,
			)
		}
	})

	t.Run("limit concurrent workers", func(t *testing.T) {
		const (
			interval  = 10 * time.Millisecond
			numWorker = 10
			maxIter   = 100

			// occasionnally we can have 1 extra concurrent goroutine,
			// we consider it an acceptable error margin
			margin             = 1
			expMaxNumGoroutine = numWorker + margin
		)

		var (
			mu               sync.Mutex
			baseNumGoroutine = runtime.NumGoroutine()
			gotNumGoroutines = make([]int, 0, maxIter)
		)

		dispatcher.New(numWorker).Do(context.Background(), maxIter, func() {
			mu.Lock()
			gotNumGoroutines = append(gotNumGoroutines, runtime.NumGoroutine()-baseNumGoroutine)
			mu.Unlock()
			time.Sleep(interval)
		})

		for _, gotNumGoroutine := range gotNumGoroutines {
			if gotNumGoroutine > expMaxNumGoroutine {
				t.Errorf(
					"max concurrent workers: exp <= %d, got %d",
					expMaxNumGoroutine, gotNumGoroutine,
				)
			}
		}

		t.Log(gotNumGoroutines)
	})

	t.Run("dispatch concurrent workers correctly", func(t *testing.T) {
		const (
			numWorker = 3
			maxIter   = 12

			minIntervalBetweenGroups = 30 * time.Millisecond
			maxIntervalWithinGroup   = 10 * time.Millisecond
		)

		var (
			// elapsedTimes is a slice of durations corresponding to the
			// intervals between the call to semimpl.Do and each callback.
			elapsedTimes = make([]time.Duration, 0, maxIter)
			mu           sync.Mutex
		)

		start := time.Now()
		dispatcher.New(numWorker).Do(context.Background(), maxIter, func() {
			mu.Lock()
			elapsedTimes = append(elapsedTimes, time.Since(start))
			mu.Unlock()
			time.Sleep(minIntervalBetweenGroups)
		})

		// check elapsedTimes slice is coherent, grouping its values
		// by expectedly similar durations, e.g.:
		// 12 iterations / 3 workers -> 4 groups of 3 similar durations.
		// With a callback duration of 30ms, we can expect such grouping:
		// [[0ms, 0ms, 0ms], [30ms, 30ms, 30ms], [60ms, 60ms, 60ms], [90ms, 90ms, 90ms]]
		// with a certain delta.
		// We check the resulting grouping against 2 rules:
		// 	1. durations within a same group must be close
		// 	2. max interval between two groups must be higher than the callback duration
		groups := groupby(elapsedTimes, numWorker)
		for groupIndex, group := range groups {
			// 1. check durations within each group are similar
			hi, lo := maxof(group), minof(group)
			if interval := hi - lo; interval > maxIntervalWithinGroup {
				t.Errorf(
					"unexpected interval in group: exp < %dms, got %dms",
					maxIntervalWithinGroup.Milliseconds(), interval.Milliseconds(),
				)
			}

			// check durations between distinct groups are spaced
			if groupIndex == len(groups)-1 {
				break
			}
			curr, next := minof(group), minof(groups[groupIndex+1])
			if interval := next - curr; interval < minIntervalBetweenGroups {
				t.Errorf(
					"unexpected interval between groups: exp > %dms, got %dms",
					minIntervalBetweenGroups.Milliseconds(), interval.Milliseconds(),
				)
			}
		}

		t.Log(elapsedTimes)
	})
}

// helpers

func groupby(src []time.Duration, by int) [][]time.Duration {
	numGroups := len(src) / by
	out := make([][]time.Duration, 0, numGroups)

	for i := 0; i < numGroups; i++ {
		lo := by * i
		hi := lo + by
		out = append(out, src[lo:hi])
	}

	return out
}

func minof(src []time.Duration) time.Duration {
	var min time.Duration
	for _, d := range src {
		if d < min || min == 0 {
			min = d
		}
	}
	return min
}

func maxof(src []time.Duration) time.Duration {
	var max time.Duration
	for _, d := range src {
		if d > max {
			max = d
		}
	}
	return max
}

func timeFunc(f func()) time.Duration {
	start := time.Now()
	f()
	return time.Since(start)
}
