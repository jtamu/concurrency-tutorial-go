package main

import "fmt"

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

	bridge := func(
		done <-chan interface{},
		chanStream <-chan <-chan interface{},
	) <-chan interface{} {
		bridgeStream := make(chan interface{})
		go func() {
			defer close(bridgeStream)
			for {
				select {
				case <-done:
					return
				case channel, ok := <-chanStream:
					if !ok {
						return
					}
					for val := range orDone(done, channel) {
						select {
						case <-done:
							return
						case bridgeStream <- val:
						}
					}
				}
			}
		}()
		return bridgeStream
	}

	genVals := func(
		done <-chan interface{},
	) <-chan <-chan interface{} {
		chanStream := make(chan (<-chan interface{}))
		go func() {
			defer close(chanStream)
			for i := 0; i < 10; i++ {
				// バッファを設定しないとデッドロックする
				stream := make(chan interface{}, 1)
				stream <- i
				chanStream <- stream
				close(stream)
			}
		}()
		return chanStream
	}

	done := make(chan interface{})
	defer close(done)

	chanStream := genVals(done)
	bridgeStream := bridge(done, chanStream)
	for val := range bridgeStream {
		fmt.Println(val)
	}
}
