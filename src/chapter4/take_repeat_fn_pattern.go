package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

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

	take := func(
		done <-chan interface{},
		valueStream <-chan interface{},
		num int,
	) <-chan interface{} {
		takeStream := make(chan interface{})
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

	done := make(chan interface{})
	defer close(done)

	rand := func() interface{} {
		n, _ := rand.Int(rand.Reader, big.NewInt(1000))
		return n
	}

	for num := range take(done, repeatFn(done, rand), 10) {
		fmt.Println(num)
	}
}
