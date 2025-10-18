package main

import (
	"fmt"
	"sync"
)

func main() {
	orDone := func(
		done <-chan interface{},
		c <-chan interface{},
	) <-chan interface{} {
		orDoneStream := make(chan interface{})
		go func() {
			defer close(orDoneStream)
			for {
				select {
				case <-done:
					return
				case ch, ok := <-c:
					if !ok {
						return
					}
					select {
					case <-done:
						return
					case orDoneStream <- ch:
					}
				}
			}
		}()
		return orDoneStream
	}

	tee := func(
		done <-chan interface{},
		in <-chan interface{},
	) (<-chan interface{}, <-chan interface{}) {
		out1 := make(chan interface{})
		out2 := make(chan interface{})

		go func() {
			defer close(out1)
			defer close(out2)

			for value := range orDone(done, in) {
				wg := sync.WaitGroup{}
				wg.Add(2)
				for _, ch := range []chan interface{}{out1, out2} {
					go func(c chan interface{}) {
						defer wg.Done()
						select {
						case <-done:
							return
						case c <- value:
						}
					}(ch)
				}
				wg.Wait()
			}
		}()
		return out1, out2
	}

	exampleChannelFn := func(
		done <-chan interface{},
	) <-chan interface{} {
		intStream := make(chan interface{})
		go func() {
			defer close(intStream)
			for i := 0; i < 5; i++ {
				select {
				case <-done:
					return
				case intStream <- i:
				}
			}
		}()
		return intStream
	}

	done := make(chan interface{})
	defer close(done)

	intStream := exampleChannelFn(done)
	out1, out2 := tee(done, intStream)
	for val1 := range out1 {
		fmt.Printf("out1: %v, out2: %v\n", val1, <-out2)
	}

	// デッドロックの例
	// 2つのチャネルは同期的に受信する必要がある
	intStream2 := exampleChannelFn(done)
	out3, out4 := tee(done, intStream2)
	for val3 := range out3 {
		fmt.Printf("out3: %v\n", val3)
	}
	for val4 := range out4 {
		fmt.Printf("out4: %v\n", val4)
	}
}
