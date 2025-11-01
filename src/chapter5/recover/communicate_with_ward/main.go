package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

type startGoroutineFn func(
	done <-chan interface{},
	pulseInterval time.Duration,
) (heartbeat <-chan interface{})

func main() {
	// or pattern from src/chapter4/goroutine_leak_fix_or_pattern.go
	var or func(channels ...<-chan interface{}) <-chan interface{}
	or = func(channels ...<-chan interface{}) <-chan interface{} {
		switch len(channels) {
		case 0:
			return nil
		case 1:
			return channels[0]
		}

		orDone := make(chan interface{})
		go func() {
			defer close(orDone)
			switch len(channels) {
			case 2:
				select {
				case <-channels[0]:
				case <-channels[1]:
				}
			default:
				select {
				case <-channels[0]:
				case <-channels[1]:
				case <-channels[2]:
				case <-or(append(channels[3:], orDone)...):
				}
			}
		}()
		return orDone
	}

	// orDone pattern from src/chapter4/or_done_pattern.go
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

	// bridge pattern from src/chapter4/bridge_channel_pattern.go
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

	// take pattern from src/chapter4/take_repeat_pattern.go
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

	newSteward := func(
		timeout time.Duration,
		startGoroutine startGoroutineFn,
	) startGoroutineFn {
		return func(
			done <-chan interface{},
			pulseInterval time.Duration,
		) <-chan interface{} {
			heartbeat := make(chan interface{})

			go func() {
				defer close(heartbeat)

				var wardDone chan interface{}
				var wardHeartbeat <-chan interface{}

				startWard := func() {
					wardDone = make(chan interface{})
					wardHeartbeat = startGoroutine(or(wardDone, done), timeout/2)
				}
				startWard()

				pulse := time.Tick(pulseInterval)

			monitorLoop:
				for {
					timeoutSignal := time.After(timeout)
					for {
						select {
						case <-pulse:
							select {
							case heartbeat <- struct{}{}:
							default:
							}
						case <-wardHeartbeat:
							continue monitorLoop
						case <-timeoutSignal:
							log.Println("steward: ward unhealty; restarting")
							close(wardDone)
							startWard()
							continue monitorLoop
						case <-done:
							return
						}
					}
				}
			}()
			return heartbeat
		}
	}

	doworkFn := func(
		done <-chan interface{},
		intList ...int,
	) (startGoroutineFn, <-chan interface{}) {
		intChanStream := make(chan (<-chan interface{}))
		intStream := bridge(done, intChanStream)

		dowork := func(
			done <-chan interface{},
			pulseInterval time.Duration,
		) <-chan interface{} {
			intStream := make(chan interface{})
			heartbeat := make(chan interface{})

			go func() {
				defer close(intStream)
				defer close(heartbeat)

				select {
				case intChanStream <- intStream:
				case <-done:
					return
				}

				pulse := time.Tick(pulseInterval)

				for {
				valueLoop:
					for _, intVal := range intList {
						if intVal < 0 {
							log.Printf("negative value: %v\n", intVal)
							return
						}

						for {
							select {
							case <-pulse:
								select {
								case heartbeat <- struct{}{}:
								default:
								}
							case intStream <- intVal:
								continue valueLoop
							case <-done:
								return
							}
						}
					}
				}
			}()
			return heartbeat
		}
		return dowork, intStream
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	done := make(chan interface{})
	defer close(done)

	doWork, intStream := doworkFn(done, 1, 2, -1, 3, 4, 5)
	doWorkWithSteward := newSteward(1*time.Millisecond, doWork)
	doWorkWithSteward(done, 1*time.Hour)

	for intVal := range take(done, intStream, 6) {
		fmt.Printf("Received: %v\n", intVal)
	}
}
