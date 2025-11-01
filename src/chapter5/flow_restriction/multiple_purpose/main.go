package main

import (
	"context"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Wait(context.Context) error
	Limit() rate.Limit
}

type multiLimiter struct {
	limiters []RateLimiter
}

func MultiLimiter(limiters ...RateLimiter) *multiLimiter {
	sort.Slice(limiters, func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	})
	return &multiLimiter{limiters: limiters}
}

func (l *multiLimiter) Wait(ctx context.Context) error {
	// 最長のリミッターの待ち時間を待つ
	// それより短いリミッターの待ち時間を待っている間に他のリミッターの待ち時間も進んでいく
	for _, limiter := range l.limiters {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *multiLimiter) Limit() rate.Limit {
	return l.limiters[0].Limit()
}

func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}

type APIConnection struct {
	apiLimiter     RateLimiter
	diskLimiter    RateLimiter
	networkLimiter RateLimiter
}

func Open() *APIConnection {
	// 1秒あたり2回のイベントを許可(バッファは1)
	secondLimit := rate.NewLimiter(Per(2, time.Second), 1)
	// 1分あたり10回のイベントを許可(バッファは10)
	minuteLimit := rate.NewLimiter(Per(10, time.Minute), 10)

	return &APIConnection{
		apiLimiter: MultiLimiter(secondLimit, minuteLimit),
		// 1秒あたり1回のイベントを許可(バッファは1)
		diskLimiter: rate.NewLimiter(rate.Limit(1), 1),
		// 1秒あたり3回のイベントを許可(バッファは3)
		networkLimiter: rate.NewLimiter(Per(3, time.Second), 3),
	}
}

func (a *APIConnection) ReadFile(ctx context.Context) error {
	if err := MultiLimiter(a.apiLimiter, a.diskLimiter).Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (a *APIConnection) ResolveAddress(ctx context.Context) error {
	if err := MultiLimiter(a.apiLimiter, a.networkLimiter).Wait(ctx); err != nil {
		return err
	}
	return nil
}

func main() {
	defer log.Printf("Done.")
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	apiConnection := Open()
	var wg sync.WaitGroup
	wg.Add(20)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ReadFile(context.Background())
			if err != nil {
				log.Printf("cannot ReadFile: %v", err)
			}
			log.Printf("ReadFile")
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ResolveAddress(context.Background())
			if err != nil {
				log.Printf("cannot ResolveAddress: %v", err)
			}
			log.Printf("ResolveAddress")
		}()
	}

	wg.Wait()
}
