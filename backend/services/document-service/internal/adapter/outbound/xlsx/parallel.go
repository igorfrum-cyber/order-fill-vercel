package xlsx

import (
	"runtime"
	"sync"
)

func workerCount(n int) int {
	if n <= 1 {
		return 1
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		return n
	}
	return workers
}

func runWorkers(n int, fn func(i int)) {
	workers := workerCount(n)
	if workers == 1 {
		for i := 0; i < n; i++ {
			fn(i)
		}
		return
	}
	var wg sync.WaitGroup
	stride := (n + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * stride
		end := start + stride
		if start >= n {
			break
		}
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(from int, to int) {
			defer wg.Done()
			for i := from; i < to; i++ {
				fn(i)
			}
		}(start, end)
	}
	wg.Wait()
}
