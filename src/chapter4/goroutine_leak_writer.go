package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func main() {
	newRandStream := func() <-chan int {
		randStream := make(chan int)
		go func() {
			defer fmt.Println("newRandStream closure exited.")
			defer close(randStream)
			for {
				n, _ := rand.Int(rand.Reader, big.NewInt(1000))
				randStream <- int(n.Int64())
			}
		}()
		return randStream
	}

	randStream := newRandStream()
	fmt.Println("3 random ints:")
	for i := 0; i < 3; i++ {
		fmt.Printf("%d: %d\n", i, <-randStream)
	}
	fmt.Println("Stopping goroutine...")
}
