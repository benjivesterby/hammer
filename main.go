package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	path := flag.String(
		"path",
		"http://127.0.0.1:8080/",
		"api request path",
	)

	method := flag.String(
		"method",
		"GET",
		"http.MethodGet flag",
	)

	p := flag.Int(
		"p",
		50,
		"number of parallel requests",
	)

	delay := flag.Duration(
		"delay",
		time.Nanosecond,
		"delay duration between requests on individual routine",
	)

	flag.Parse()

	ctx := initContext()

	var total time.Duration
	count := 0
	var mu sync.Mutex

	// Create pool
	for i := 0; i < *p; i++ {

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					delta, ok := request(*method, *path)
					if !ok {
						time.Sleep(*delay)
						continue
					}

					mu.Lock()
					total += delta
					count++
					mu.Unlock()

					time.Sleep(*delay)
				}
			}
		}()
	}

	var avg int
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mu.Lock()
			total := total
			count := count
			mu.Unlock()

			if count > 0 {
				avg = int(total) / count
			}

			fmt.Printf("total transactions: %v - average transaction time: %v\n", count, time.Duration(avg))
			time.Sleep(time.Second * 5)
		}
	}
}

func request(method, path string) (time.Duration, bool) {
	client := &http.Client{}
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return 0, false
	}

	start := time.Now()
	resp, err := client.Do(req)

	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return 0, false
	}

	return time.Since(start), true
}

func initContext() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	// Setup interrupt monitoring for the agent
	go func() {
		defer cancel()

		select {
		case <-ctx.Done():
			return
		case <-sigs:
			fmt.Println("exiting syncer")
			os.Exit(1)
		}
	}()

	return ctx
}
