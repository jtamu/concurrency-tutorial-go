package main

import (
	"fmt"
	"sync"
)

func main() {
	var numCalcsCreated int
	var numCalcsMutex sync.Mutex

	calcPool := &sync.Pool{
		New: func() interface{} {
			numCalcsMutex.Lock()
			numCalcsCreated++
			numCalcsMutex.Unlock()
			mem := make([]byte, 1024)
			return &mem
		},
	}

	calcPool.Put(calcPool.New())
	calcPool.Put(calcPool.New())
	calcPool.Put(calcPool.New())

	const numWorkers = 1024 * 1024

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := numWorkers; i > 0; i-- {
		go func() {
			defer wg.Done()

			mem := calcPool.Get().(*[]byte)
			defer calcPool.Put(mem)

			// Do something with the memory
		}()
	}

	wg.Wait()
	fmt.Printf("%d calculators were created.\n", numCalcsCreated)
}
