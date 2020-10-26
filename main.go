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

	file := flag.String(
		"f",
		"",
		"output file name, with format as extension [csv,json]",
	)

	flag.Parse()

	ctx := initContext()

	var out chan stat
	var foutput bool
	if *file != "" {
		out = make(chan stat)
		foutput = true
	}

	var total time.Duration
	count := 0
	var mu sync.Mutex

	// Create pool
	for i := 0; i < *p; i++ {

		go func(worker int, output bool) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.Tick(*delay):
					delta, ok := request(*method, *path)
					if !ok {
						time.Sleep(*delay)
						continue
					}

					var req int
					mu.Lock()
					total += delta
					count++
					req = count
					mu.Unlock()

					if output {
						select {
						case <-ctx.Done():
							return
						case out <- stat{
							Request: req,
							Worker:  worker,
							Success: ok,
							Elapsed: delta,
						}:
						}
					}
				}
			}
		}(i, foutput)
	}

	if foutput {
		_, err := os.Stat(*file)
		if err != nil {
			_, err = os.Create(*file)
			if err != nil {
				fmt.Printf("error writing to file: %s\n", err.Error())
			}
		}

		f, err := os.OpenFile(*file, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			fmt.Printf("error writing to file: %s\n", err.Error())
		}

		go func(out <-chan stat) {
			defer func() {
				_ = f.Close()
			}()

			for {

				select {
				case <-ctx.Done():
					return
				case data, ok := <-out:
					if !ok {
						return
					}

					line := fmt.Sprintln(data.String())
					_, err := f.WriteString(line)
					if err != nil {
						fmt.Printf("error writing to file: %s\n", err.Error())
					}
				}
			}

		}(out)

	}

	var avg int
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(time.Second * 5):
			mu.Lock()
			total := total
			count := count
			mu.Unlock()

			if count > 0 {
				avg = int(total) / count
			}

			fmt.Printf("total transactions: %v - average transaction time: %v\n", count, time.Duration(avg))
		}
	}
}

func request(method, path string) (time.Duration, bool) {
	client := &http.Client{}
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return time.Duration(0), false
	}

	start := time.Now()
	resp, err := client.Do(req)

	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return time.Duration(0), false
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
