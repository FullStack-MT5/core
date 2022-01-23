package semimpl

import (
	"context"
	"errors"
	"log"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Do concurrently executes callback at most maxIter times or until ctx is done
// or canceled. Concurrency is handled leveraging the semaphore pattern, which
// ensures at most numWorkers goroutines are spawned at se same time.
func Do(ctx context.Context, numWorkers, maxIter int, callback func()) {
	sem := semaphore.NewWeighted(int64(numWorkers))
	wg := sync.WaitGroup{}

	for i := 0; i < maxIter || maxIter == 0; i++ {
		wg.Add(1)

		if err := sem.Acquire(ctx, 1); err != nil {
			handleContextError(err)
			wg.Done()
			break
		}

		go func() {
			defer func() {
				sem.Release(1)
				wg.Done()
			}()
			callback()
		}()
	}

	wg.Wait()
}

func handleContextError(err error) {
	switch {
	case err == nil:
	case errors.Is(err, context.DeadlineExceeded):
	case errors.Is(err, context.Canceled):
	default:
		log.Fatal(err)
	}
}
