package main

import "fmt"

func main() {
	orDone := func(
		done <-chan interface{},
		c <-chan int,
	) <-chan int {
		orDoneStream := make(chan int)
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

	exampleChannelFn := func(
		done <-chan interface{},
	) <-chan int {
		intStream := make(chan int)
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
	for integer := range orDone(done, intStream) {
		fmt.Printf("%v\n", integer)
	}
}
