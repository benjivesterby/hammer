package main

import (
	"fmt"
	"time"
)

type stat struct {
	Request int
	Worker  int
	Success bool
	Elapsed time.Duration
}

func (s stat) String() string {
	return fmt.Sprintf(
		"%v,%v,%v,%v",
		s.Request,
		s.Worker,
		s.Success,
		s.Elapsed,
	)
}
