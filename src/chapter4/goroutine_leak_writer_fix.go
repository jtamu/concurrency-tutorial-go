package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

func main() {
	newRandStream := func(done <-chan interface{}) <-chan int {
		randStream := make(chan int)
		go func() {
			defer fmt.Println("newRandStream closure exited.")
			defer close(randStream)
			for {
				n, _ := rand.Int(rand.Reader, big.NewInt(1000))
				select {
				case <-done:
					return
				// case文に書かないとブロックされる
				case randStream <- int(n.Int64()):
				}
			}
		}()
		return randStream
	}

	done := make(chan interface{})
	randStream := newRandStream(done)
	fmt.Println("3 random ints:")
	for i := 0; i < 3; i++ {
		fmt.Printf("%d: %d\n", i, <-randStream)
	}
	fmt.Println("Stopping goroutine...")
	close(done)

	// Wait for the goroutine to finish
	time.Sleep(1 * time.Second)
}
