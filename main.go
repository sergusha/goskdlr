package main

import (
	"context"
	"fmt"
	"github.com/sergusha/goskdlr/goskdlr"
	"os"
	"time"
)

func foo(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				fmt.Println("foo is Stopped at", time.Now())
				return
			case context.Canceled:
				fmt.Println("foo is Stopped at", time.Now())
				return
			}
		default:
			fmt.Println("foo is working at", time.Now())
			time.Sleep(time.Second * 2)
		}
	}
}

func bar(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				fmt.Println("bar is Stopped at", time.Now())
				return
			case context.Canceled:
				fmt.Println("bar is Stopped at", time.Now())
				return
			}
		default:
			fmt.Println("bar is working at", time.Now())
			time.Sleep(time.Second * 2)
		}
	}
}

func main() {
	s, err := goskdlr.New("1s")
	if err != nil {
		os.Exit(1)
	}
	ctx1 := context.TODO()

	fooSchedule := time.Now().Add(time.Second)
	fmt.Println("foo is scheduled for", fooSchedule)
	//	fooPeriod := time.Second
	j, err := goskdlr.NewJob(ctx1, &fooSchedule, nil, foo)
	if err != nil {
		fmt.Println("coundn't create Job")
	}
	s.ScheduleJob(j)
	ctx2 := context.TODO()
	barSchedule := time.Now().Add(time.Second * 5)
	fmt.Println("bar is scheduled for", barSchedule)
	k, err := goskdlr.NewJob(ctx2, &barSchedule, nil, bar)
	if err != nil {
		fmt.Println("couldn't create Job")
	}
	s.ScheduleJob(k)
	fmt.Println("Working...")
	time.Sleep(time.Second * 10)
	s.CancelJob(j)
	s.Stop()
	time.Sleep(time.Second * 15)
}
