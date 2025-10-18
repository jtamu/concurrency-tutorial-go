package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// 何故かファンアウト未対応版よりも時間がかかる

func main() {
	repeatFn := func(
		done <-chan interface{},
		fn func() interface{},
	) <-chan interface{} {
		repeatStream := make(chan interface{})
		go func() {
			defer close(repeatStream)
			for {
				select {
				case <-done:
					return
				case repeatStream <- fn():
				}
			}
		}()
		return repeatStream
	}

	toInt := func(
		done <-chan interface{},
		valueStream <-chan interface{},
	) <-chan int {
		intStream := make(chan int)
		go func() {
			defer close(intStream)
			for v := range valueStream {
				select {
				case <-done:
					return
				case intStream <- v.(int):
				}
			}
		}()
		return intStream
	}

	takeInt := func(
		done <-chan interface{},
		valueStream <-chan int,
		num int,
	) <-chan int {
		takeStream := make(chan int)
		go func() {
			defer close(takeStream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	isPrime := func(integer int) bool {
		for i := 2; i < integer; i++ {
			if integer%i == 0 {
				return false
			}
		}
		return true
	}

	primeFinder := func(
		done <-chan interface{},
		intStream <-chan int,
	) <-chan int {
		output := make(chan int)
		go func() {
			defer close(output)
			for {
				select {
				case <-done:
					return
				case integer := <-intStream:
					if isPrime(integer) {
						output <- integer
					}
				}
			}
		}()
		return output
	}

	rand := func() interface{} {
		return rand.Intn(500000000)
	}

	fanIn := func(
		done <-chan interface{},
		channels ...<-chan int,
	) <-chan int {
		multiplexedStream := make(chan int)
		go func() {
			defer close(multiplexedStream)

			wg := sync.WaitGroup{}
			wg.Add(len(channels))
			for _, channel := range channels {
				go func() {
					defer wg.Done()
					for {
						select {
						case <-done:
							return
						case ch, ok := <-channel:
							if !ok {
								return
							}
							multiplexedStream <- ch
						}
					}
				}()
			}
			wg.Wait()
		}()
		return multiplexedStream
	}

	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	randIntStream := toInt(done, repeatFn(done, rand))
	fmt.Println("Primes:")

	numFinders := runtime.NumCPU()
	finders := make([]<-chan int, numFinders)
	for i := 0; i < numFinders; i++ {
		finders[i] = primeFinder(done, randIntStream)
	}
	fanInFinders := fanIn(done, finders...)

	for prime := range takeInt(done, fanInFinders, 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v\n", time.Since(start))
}
